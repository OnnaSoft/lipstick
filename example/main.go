package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	r.Use(gin.Logger())

	r.GET("/", func(ctx *gin.Context) {
		ctx.Status(200)
		fmt.Fprint(ctx.Writer, "hello")
		ctx.Writer.Flush()
		fmt.Fprint(ctx.Writer, "world")
	})

	r.Run(":12000")
}
