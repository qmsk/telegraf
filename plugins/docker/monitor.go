package docker

import (
    "github.com/fsouza/go-dockerclient"
    "fmt"
    "log"
    "os"
)

// monitored container state
type monitorContainer struct {
    name        string
    log         *log.Logger
    tags         map[string]string
    statsChan    chan *docker.Stats
    stats        *docker.Stats
}

func (self monitorContainer) String() string {
    return self.name
}

// call as a new goroutine to start monitoring stats
// closes statsChan on errors
func (self *monitorContainer) start(dockerClient *docker.Client, listContainer docker.APIContainers) {
    statsOptions := docker.StatsOptions{
        ID:     listContainer.ID,
        Stats:  self.statsChan,
        Stream: true,
    }

    self.log.Printf("Start %#v...\n", statsOptions)

    // this is a blocking operation
    if err := dockerClient.Stats(statsOptions); err != nil {
        self.log.Printf("Error: %v\n", err)
    } else {
        self.log.Printf("End\n")
    }
}

// maintain latests stats
func (self *monitorContainer) run() {
    for dockerStats := range self.statsChan {
        self.log.Printf("%v\n", dockerStats)

        self.stats = dockerStats
    }

    self.stats = nil
}

// starts monitoring a container's stats
func newMonitorContainer(dockerClient *docker.Client, listContainer docker.APIContainers) *monitorContainer {
    containerName := listContainer.Names[0][1:]

    monitorContainer := &monitorContainer{
        name:       containerName,
        log:        log.New(os.Stderr, fmt.Sprintf("plugins/docker %s: ", listContainer.ID), 0),
        tags:       map[string]string{
            "id":       listContainer.ID,
            "image":    listContainer.Image,
            "name":     containerName,
        },
        statsChan:  make(chan *docker.Stats),
    }

    go monitorContainer.start(dockerClient, listContainer)
    go monitorContainer.run()

    return monitorContainer
}
