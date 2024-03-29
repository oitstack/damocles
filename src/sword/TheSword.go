package sword

import (
	"bufio"
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"gopkg.in/matryer/try.v1"
	"log"
	"math/big"
	"net"
	"sync"
	"time"
)

type ITheSword interface {
	Start()
}

type TheSword struct {
	Cli     client.Client
	Target  string
	Timeout big.Int
}

func NewTheSword(target string, timeout big.Int) *TheSword {
	if cli, err := client.NewEnvClient(); err != nil {
		panic(err)
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err = cli.Ping(ctx); err != nil {
			panic(err)
		} else {
			return &TheSword{Target: target, Timeout: timeout, Cli: *cli}
		}
	}

}

func (theSword *TheSword) Start() {
	//1、开始监听
	reached := make(chan bool, 1)
	var wg sync.WaitGroup
	go theSword.welcomeToDamocles(reached, theSword.Timeout, &wg)
	//2、若长时间没连接则退出
	select {
	case <-time.After(time.Duration(theSword.Timeout.Uint64()) * time.Second):
		panic("no damocles has reach in 30 sec.")
	case <-reached:
		log.Println("damocles has come")
	}
	wg.Wait()
	//3.等待所有客户端断连，则开始清理
	log.Println("killing men")
	theSword.hit()
	log.Println("the job has done")
}

func (theSword *TheSword) welcomeToDamocles(reached chan<- bool, timeout big.Int, wg *sync.WaitGroup) {
	var once sync.Once
	if li, err := net.Listen("tcp", "0.0.0.0:8080"); err != nil {
		panic(err)
	} else {
		if conn, err := li.Accept(); err != nil {
			panic(err)
		} else {
			go theSword.ensureAlive(once, reached, conn, wg)

		}
	}
}
func (theSword TheSword) ensureAlive(once sync.Once, reached chan<- bool, conn net.Conn, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()
	once.Do(func() {
		reached <- true
	})

	reader := bufio.NewReader(conn)
	for {
		conn.SetReadDeadline(time.Now().Add(20 * time.Second))
		if command, err := reader.ReadString('\n'); err != nil {
			log.Printf("received an error command %s", err)
			break
		} else {
			conn.Write([]byte(command))
		}
	}
	log.Printf("connection has disconnect.")
}

func (theSword *TheSword) hit() {

	deletedContainers := make(map[string]bool)
	pruneContainers := make(map[string]bool)
	deletedNetworks := make(map[string]bool)
	deletedVolumes := make(map[string]bool)
	deletedImages := make(map[string]bool)

	args := filters.NewArgs()

	args.Add("label", theSword.Target)

	log.Printf("killing %s", args)

	if containers, err := theSword.Cli.ContainerList(context.Background(), types.ContainerListOptions{All: true, Filters: args}); err != nil {
		log.Println(err)
	} else {
		for _, container := range containers {
			theSword.Cli.ContainerRemove(context.Background(), container.ID, types.ContainerRemoveOptions{RemoveVolumes: true, Force: true})
			deletedContainers[container.ID] = true
			log.Printf("container killed: %s", container.ID)
		}
	}

	pruneFilters := filters.NewArgs()
	pruneFilters.Add("label", "theSword=true")
	try.Do(func(attempt int) (bool, error) {
		containerPruneReport, err := theSword.Cli.ContainersPrune(context.Background(), pruneFilters)
		for _, id := range containerPruneReport.ContainersDeleted {
			pruneContainers[id] = true
			log.Printf("container pruned: %s", id)
		}

		shouldRetry := attempt < 10
		if err != nil && shouldRetry {
			log.Printf("Network pruning has failed, retrying(%d/%d). The error was: %v", attempt, 10, err)
			time.Sleep(1 * time.Second)
		}
		return shouldRetry, err
	})

	try.Do(func(attempt int) (bool, error) {
		networksPruneReport, err := theSword.Cli.NetworksPrune(context.Background(), args)
		for _, networkID := range networksPruneReport.NetworksDeleted {
			deletedNetworks[networkID] = true
		}
		shouldRetry := attempt < 10
		if err != nil && shouldRetry {
			log.Printf("Network pruning has failed, retrying(%d/%d). The error was: %v", attempt, 10, err)
			time.Sleep(1 * time.Second)
		}
		return shouldRetry, err
	})

	try.Do(func(attempt int) (bool, error) {
		volumesPruneReport, err := theSword.Cli.VolumesPrune(context.Background(), args)
		for _, volumeName := range volumesPruneReport.VolumesDeleted {
			deletedVolumes[volumeName] = true
		}
		shouldRetry := attempt < 10
		if err != nil && shouldRetry {
			log.Printf("Volumes pruning has failed, retrying(%d/%d). The error was: %v", attempt, 10, err)
			time.Sleep(1 * time.Second)
		}
		return shouldRetry, err
	})

	//imagePrunedFilters := filters.NewArgs()
	//imagePrunedFilters.Add("dangling", "true")
	//try.Do(func(attempt int) (bool, error) {
	//	imagesPruneReport, err := theSword.Cli.ImagesPrune(context.Background(), imagePrunedFilters)
	//	for _, image := range imagesPruneReport.ImagesDeleted {
	//		deletedImages[image.Deleted] = true
	//	}
	//	shouldRetry := attempt < 10
	//	if err != nil && shouldRetry {
	//		log.Printf("Images pruning has failed, retrying(%d/%d). The error was: %v", attempt, 10, err)
	//		time.Sleep(1 * time.Second)
	//	}
	//	return shouldRetry, err
	//})
	log.Printf("Removed %d container(s), %d network(s), %d volume(s) %d image(s) , Pruned %d container(s)", len(deletedContainers), len(deletedNetworks), len(deletedVolumes), len(deletedImages), len(pruneContainers))
}
