package main

import (
	"archive/tar"
	"github.com/dutchcoders/goftp"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/substitutes/push-backend/util"
	"gopkg.in/alecthomas/kingpin.v2"
	"io"
	"strconv"
	"time"
)

var (
	ftpHost      = kingpin.Flag("ftp-host", "Hostname of the FTP server").Short('s').Required().String()
	ftpUser      = kingpin.Flag("ftp-user", "Username for the given FTP host").Short('u').Required().String()
	ftpPass      = kingpin.Flag("ftp-password", "Password for the given FTP host").Short('p').Required().String()
	ftpDir       = kingpin.Flag("ftp-directory", "Specify a directory where to upload the received files").Short('d').Default(".").String()
	port         = kingpin.Flag("port", "Port for the web server").Default("3000").Int()
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
	kingpin.Parse()

	r := gin.Default()

	// Connect to the FTP server
	conn, err := goftp.Connect(*ftpHost)
	if err != nil {
		log.Fatal("Failed to dial FTP server: ", err)
	}
	if err := conn.Login(*ftpUser, *ftpPass); err != nil {
		log.Fatal("Failed to authenticate against FTP server: ", err)
	}

	if *ftpDir != "." {
		if err := conn.Mkd(*ftpDir); err != nil {
			log.Fatal("Failed to create given directory ", err)
		}
	}

	r.GET("/", func(c *gin.Context) {
		c.String(200, "OK")
	})

	apiAuth := r.Group("/api", gin.BasicAuth(gin.Accounts{*authUser: *authPassword}))
	{
		apiAuth.POST("/push", func(c *gin.Context) {
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
			r := tar.NewReader(file)
			defer file.Close()
			var uploaded []FileUpload
			for {
				file, err := r.Next()
				if err == io.EOF {
					break
				}
				if err != nil {
					c.JSON(500, util.NewError("Could not read tar archive contents", err))
					return
				}
				if err := conn.Stor(file.Name, r); err != nil {
					c.JSON(500, util.NewError("Could not upload file to FTP server", err))
					return
				}
				uploaded = append(uploaded, FileUpload{Name: file.Name, Size: file.Size, UploadedAt: time.Now().Unix()})
			}
			c.JSON(200, uploaded)
		})

		apiAuth.GET("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "OK"})
		})
	}

	r.Run(":" + strconv.Itoa(*port))
}
