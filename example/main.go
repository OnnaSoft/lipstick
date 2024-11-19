package main

import (
	"fmt"
	"io"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	r.GET("/", func(ctx *gin.Context) {
		ctx.Status(200)
		fmt.Fprint(ctx.Writer, "hello ")
		ctx.Writer.Flush()
		fmt.Fprint(ctx.Writer, "world")
	})
	r.POST("/", func(ctx *gin.Context) {
		io.Copy(os.Stdout, ctx.Request.Body)
		fmt.Println()
		ctx.Status(200)
		fmt.Fprint(ctx.Writer, "hello ")
		ctx.Writer.Flush()
		fmt.Fprint(ctx.Writer, "world")
	})

	err := r.Run(":12000")
	if err != nil {
		fmt.Println(err)
	}
}
