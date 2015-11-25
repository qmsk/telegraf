package docker

import (
    //    "github.com/fsouza/go-dockerclient"
    "github.com/influxdb/telegraf/plugins"
)

type Docker struct {

}

func (self *Docker) Description() string {
    return "Gather docker container stats"
}

func (self *Docker) SampleConfig() string {
    return ""
}

func (self *Docker) Gather(acc plugins.Accumulator) error {
    return nil
}

func init() {
    plugins.Add("docker-stats", func() plugins.Plugin {
        return &Docker{}
    })
}
