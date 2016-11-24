package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
)

// StartServer start a web server
func StartServer(port string, mode string) error {
	gin.SetMode(mode)

	router := gin.New()

	// Global middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	router.GET("/cluster", getClusterInfo)

	daemonsGroup := router.Group("/daemons")
	{
		daemonsGroup.GET("", getDaemonHandler)
		daemonsGroup.POST("", postDaemonHandler)
		daemonsGroup.DELETE(":taskARN", deleteDaemonHandler)
	}

	return router.Run(":" + port)
}

func getClusterInfo(c *gin.Context) {
	res := State.FindClusterByName(Cluster)
	c.JSON(http.StatusOK, gin.H{
		"error": false,
		"data":  res,
	})
}

func getDaemonHandler(c *gin.Context) {
	// TODO Implement function to show the info of given daemon
	c.JSON(http.StatusOK, gin.H{
		"error": false,
		"data":  "",
	})
}

func postDaemonHandler(c *gin.Context) {
	taskDefinition := c.PostForm("taskDefinition")

	c.JSON(http.StatusAccepted, gin.H{
		"error": false,
	})

	// TODO Implement the function that register a batch job to lunch task on each host every 5 mins
	err := StartTask([]string{taskDefinition})
	if err != nil {
		glog.Warningln(err.Error())
	}
}

func deleteDaemonHandler(c *gin.Context) {
	// TODO Implement deleteTask function
	// Delete the task
	// deleteTask(c.Param("taskARN"))

	c.JSON(http.StatusOK, gin.H{
		"error":               false,
		"task-definition-arn": "",
	})

}
