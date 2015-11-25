package docker

import (
    "github.com/fsouza/go-dockerclient"
    "log"
    "os"
    "github.com/influxdb/telegraf/plugins"
)

type Docker struct {
    log *log.Logger
}

func newDocker() plugins.Plugin {
    return &Docker{
        log:    log.New(os.Stderr, "plugins/docker: ", 0),
    }
}

func (self *Docker) Description() string {
    return "Gather docker container stats"
}

func (self *Docker) SampleConfig() string {
    return ""
}

func (self *Docker) dockerClient() (*docker.Client, error) {
    self.log.Printf("ENV DOCKER_CERT_PATH=%s\n", os.Getenv("DOCKER_CERT_PATH"))

    if dockerClient, err := docker.NewClientFromEnv(); err != nil {
        return nil, err
    } else {
        self.log.Printf("docker.Client %+v\n", dockerClient)

        return dockerClient, nil
    }
}

func (self *Docker) gatherStats(acc plugins.Accumulator, listContainer docker.APIContainers, statsChan chan *docker.Stats) {
    // gath
    containerTags := map[string]string{
            "id":       listContainer.ID,
            "image":    listContainer.Image,
            "name":     listContainer.Names[0][1:],
    }

    for dockerStats := range statsChan {
        self.log.Printf("gatherStats %s: %v\n", listContainer.ID, dockerStats)

        memoryFields := map[string]interface{}{
            "usage":    dockerStats.MemoryStats.Usage,
        }

        acc.AddFields("memory", memoryFields, containerTags, dockerStats.Read)
    }
}

func (self *Docker) Gather(acc plugins.Accumulator) error {
    dockerClient, err := self.dockerClient()
    if err != nil {
        return err
    }

    listContainers, err := dockerClient.ListContainers(docker.ListContainersOptions{})
    if err != nil {
        return err
    }
    for _, listContainer := range listContainers {
        statsChan := make(chan *docker.Stats)
        statsOptions := docker.StatsOptions{
            ID:     listContainer.ID,
            Stats:  statsChan,
            Stream: false,
        }

        self.log.Printf("Gather container %v...\n", listContainer.ID)
        go func() {
            if err := dockerClient.Stats(statsOptions); err != nil {
                close(statsChan)
            }
        }()

        self.gatherStats(acc, listContainer, statsChan)
    }

    return nil
}

func init() {
    plugins.Add("docker2", newDocker)
}
