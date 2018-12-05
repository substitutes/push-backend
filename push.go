package main

import (
	"github.com/gin-gonic/gin"
	"github.com/jlaffaye/ftp"
	log "github.com/sirupsen/logrus"
	"github.com/substitutes/push-backend/util"
	"gopkg.in/alecthomas/kingpin.v2"
	"strconv"
	"time"
)

var (
	ftpHost      = kingpin.Flag("ftp-host", "Hostname of the FTP server").Short('s').Required().String()
	ftpUser      = kingpin.Flag("ftp-user", "Username for the given FTP host").Short('u').Required().String()
	ftpPass      = kingpin.Flag("ftp-password", "Password for the given FTP host").Short('p').Required().String()
	ftpDir       = kingpin.Flag("ftp-directory", "Specify a directory where to upload the received files").Short('d').Default(".").String()
	port         = kingpin.Flag("port", "Port for the web server").Default("3000").Int()
	verbose      = kingpin.Flag("verbose", "Enable verbose output").Short('v').Bool()
	quiet        = kingpin.Flag("quiet", "Only display warnings").Short('q').Bool()
	authUser     = kingpin.Flag("username", "Username for HTTP basic auth").Default("substitutes").String()
	authPassword = kingpin.Flag("password", "Password for HTTP basic auth").Default("substitutes").String()
)

type FileUpload struct {
	UploadedAt int64  `json:"uploaded_at"`
	Name       string `json:"name"`
	Size       int64  `json:"size"`
}

func dir(s string) string { return *ftpDir + s }

func main() {
	kingpin.CommandLine.HelpFlag.Short('h')
	kingpin.Parse()

	if *verbose {
		log.SetLevel(log.DebugLevel)
	} else if *quiet {
		log.SetLevel(log.WarnLevel)
	}
	r := gin.Default()

	conn := connect()

	if *ftpDir != "." {
		if err := conn.MakeDir(*ftpDir); err != nil {
			log.Fatal("Failed to create given directory: ", err)
		}
		if err := conn.ChangeDir(*ftpDir); err != nil {
			log.Fatal("Failed to change to given directory: ", err)
		}
	}

	r.GET("/", func(c *gin.Context) {
		c.String(200, "OK")
	})

	apiAuth := r.Group("/api", gin.BasicAuth(gin.Accounts{*authUser: *authPassword}))
	{
		apiAuth.POST("/push", func(c *gin.Context) {
			// Reconnect to FTP
			conn = connect()
			// Get the data
			f, err := c.FormFile("push")
			if err != nil {
				c.JSON(500, util.NewError("Could not get file", err))
				return
			}
			file, err := f.Open()
			if err != nil {
				c.JSON(400, util.NewError("Could not open file", err))
				return
			}
			var uploaded []FileUpload
			if err := conn.Stor(f.Filename, file); err != nil {
				c.JSON(500, util.NewError("Could not upload file to FTP server", err))
				return
			} else {
				uploaded = append(uploaded, FileUpload{Name: f.Filename, Size: f.Size, UploadedAt: time.Now().Unix()})
			}
			file.Close()
			c.JSON(201, uploaded)
		})

		apiAuth.GET("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "OK"})
		})
	}

	r.Run(":" + strconv.Itoa(*port))
}

func connect() *ftp.ServerConn {
	// Connect to the FTP server
	conn, err := ftp.Connect(*ftpHost)
	if err != nil {
		log.Fatal("Failed to dial FTP server: ", err)
	}
	defer conn.Logout()
	if err := conn.Login(*ftpUser, *ftpPass); err != nil {
		log.Fatal("Failed to authenticate against FTP server: ", err)
	}

	return conn
}
