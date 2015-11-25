package docker

import (
    "github.com/fsouza/go-dockerclient"
    "log"
    "os"
    "github.com/influxdb/telegraf/plugins"
)

type Docker struct {
    log *log.Logger

    containers map[string]*monitorContainer
}

func newDocker() plugins.Plugin {
    return &Docker{
        log:        log.New(os.Stderr, "plugins/docker: ", 0),
        containers: make(map[string]*monitorContainer),
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

// maintain the list of monitored containers
// self.containers should be an up-to-date list of monitored containers
func (self *Docker) updateContainers() error {
    dockerClient, err := self.dockerClient()
    if err != nil {
        return err
    }

    listContainers, err := dockerClient.ListContainers(docker.ListContainersOptions{})
    if err != nil {
        return err
    }

    // add new containers
    for _, listContainer := range listContainers {
        if _, exists := self.containers[listContainer.ID]; !exists {
            monitorContainer := newMonitorContainer(dockerClient, listContainer)

            self.log.Printf("New container: %v\n", monitorContainer)

            self.containers[listContainer.ID] = monitorContainer

            // XXX: wait
            for monitorContainer.stats == nil {

            }
        }
    }

    // cleanup dead containers
    for containerID, monitorContainer := range self.containers {
        if monitorContainer.stats == nil {
            self.log.Printf("Drop container: %v\n", monitorContainer)
            delete(self.containers, containerID)
        }
    }

    return nil
}

func (self *Docker) Gather(acc plugins.Accumulator) error {
    if err := self.updateContainers(); err != nil {
        return err
    }

    // gather
    for _, monitorContainer := range self.containers {
        // XXX: safe?
        dockerStats := monitorContainer.stats

        if dockerStats == nil {
            continue
        }

        acc.AddFields("network", map[string]interface{}{
            "rx_bytes":     dockerStats.Network.RxBytes,
            "rx_dropped":   dockerStats.Network.RxDropped,
            "rx_errors":    dockerStats.Network.RxErrors,
            "rx_packets":   dockerStats.Network.RxPackets,
            "tx_bytes":     dockerStats.Network.TxBytes,
            "tx_dropped":   dockerStats.Network.TxDropped,
            "tx_errors":    dockerStats.Network.TxErrors,
            "tx_packets":   dockerStats.Network.TxPackets,
        }, monitorContainer.tags, dockerStats.Read)

        acc.AddFields("memory", map[string]interface{}{
            "cache":        dockerStats.MemoryStats.Stats.Cache,
            "rss":          dockerStats.MemoryStats.Stats.Rss,
            "max_usage":    dockerStats.MemoryStats.MaxUsage,
            "usage":        dockerStats.MemoryStats.Usage,
            "failcnt":      dockerStats.MemoryStats.Failcnt,
            "limit":        dockerStats.MemoryStats.Limit,
        }, monitorContainer.tags, dockerStats.Read)

        acc.AddFields("cpu", monitorContainer.GatherCPU(), monitorContainer.tags, dockerStats.Read)
    }

    return nil
}

func init() {
    plugins.Add("docker2", newDocker)
}
