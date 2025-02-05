package cmd

import (
	"context"
	"fmt"

	"github.com/go-cmd/cmd"
	"github.com/kubeshark/kubeshark/config"
	"github.com/kubeshark/kubeshark/kubernetes"
	"github.com/kubeshark/kubeshark/utils"
	"github.com/rivo/tview"
	"github.com/rs/zerolog/log"
)

func runPprof() {
	runProxy(false, true)

	provider, err := getKubernetesProviderForCli(false, false)
	if err != nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pods, err := provider.ListPodsByAppLabel(ctx, config.Config.Tap.Release.Namespace, map[string]string{"app.kubeshark.co/app": "worker"})
	if err != nil {
		log.Error().
			Err(err).
			Msg("Failed to list worker pods!")
		cancel()
		return
	}

	fullscreen := true

	app := tview.NewApplication()
	list := tview.NewList()

	var currentCmd *cmd.Cmd

	i := 48
	for _, pod := range pods {
		for _, container := range pod.Spec.Containers {
			log.Info().Str("pod", pod.Name).Str("container", container.Name).Send()
			homeUrl := fmt.Sprintf("%s/pprof/%s/%s/", kubernetes.GetHubUrl(), pod.Status.HostIP, container.Name)
			modal := tview.NewModal().
				SetText(fmt.Sprintf("pod: %s container: %s", pod.Name, container.Name)).
				AddButtons([]string{
					"Open Debug Home Page",
					"Profile: CPU",
					"Profile: Memory",
					"Profile: Goroutine",
					"Cancel",
				}).
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					switch buttonLabel {
					case "Open Debug Home Page":
						utils.OpenBrowser(homeUrl)
					case "Profile: CPU":
						if currentCmd != nil {
							err = currentCmd.Stop()
							if err != nil {
								log.Error().Err(err).Str("name", currentCmd.Name).Msg("Failed to stop process!")
							}
						}
						currentCmd = cmd.NewCmd("go", "tool", "pprof", "-http", ":8000", fmt.Sprintf("%sprofile", homeUrl))
						currentCmd.Start()
					case "Profile: Memory":
						if currentCmd != nil {
							err = currentCmd.Stop()
							if err != nil {
								log.Error().Err(err).Str("name", currentCmd.Name).Msg("Failed to stop process!")
							}
						}
						currentCmd = cmd.NewCmd("go", "tool", "pprof", "-http", ":8000", fmt.Sprintf("%sheap", homeUrl))
						currentCmd.Start()
					case "Profile: Goroutine":
						if currentCmd != nil {
							err = currentCmd.Stop()
							if err != nil {
								log.Error().Err(err).Str("name", currentCmd.Name).Msg("Failed to stop process!")
							}
						}
						currentCmd = cmd.NewCmd("go", "tool", "pprof", "-http", ":8000", fmt.Sprintf("%sgoroutine", homeUrl))
						currentCmd.Start()
					case "Cancel":
						if currentCmd != nil {
							err = currentCmd.Stop()
							if err != nil {
								log.Error().Err(err).Str("name", currentCmd.Name).Msg("Failed to stop process!")
							}
						}
						fallthrough
					default:
						app.SetRoot(list, fullscreen)
					}
				})
			list.AddItem(fmt.Sprintf("pod: %s container: %s", pod.Name, container.Name), pod.Spec.NodeName, rune(i), func() {
				app.SetRoot(modal, fullscreen)
			})
			i++
		}
	}

	list.AddItem("Quit", "Press to exit", 'q', func() {
		if currentCmd != nil {
			err = currentCmd.Stop()
			if err != nil {
				log.Error().Err(err).Str("name", currentCmd.Name).Msg("Failed to stop process!")
			}
		}
		app.Stop()
	})

	if err := app.SetRoot(list, fullscreen).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
