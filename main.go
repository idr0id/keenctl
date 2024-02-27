package main

import (
	"fmt"
	"github.com/docopt/docopt-go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/schollz/progressbar/v3"
	"os"
	"slices"
	"sync"
)

var version = "[dev-build]"

const usage = `routek - keep actual static routes on keenetic's router.

Usage:
  routek [options] [-v]...

Options:
  -c --config <path>      Path to the configuration file.
                           [default: ./config.toml]
  -v --verbose            Print debug information on stderr.
  -h --help               Show this help.
`

func main() {
	args := parseArgs()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	parseVerbosity(args)

	rootCfg := loadConfig(args)
	log.Trace().Interface("config", rootCfg).Msg("configuration loaded")

	log.Info().Msg("load static routes")
	provisionAddressesMap := parseStaticRoutesCfg(rootCfg)
	provisionAddressesAll := flatAddresses(provisionAddressesMap)
	log.Info().Msgf("loaded %d static routes", len(provisionAddressesAll))

	for name, routerCfg := range rootCfg.Routers {
		l := log.With().Str("router", name).Logger()
		addr := fmt.Sprintf("%s:%d", routerCfg.Host, routerCfg.Port)

		sshClients := make(chan *SshClient, routerCfg.MaxParallelCommands)
		for i := 0; i < routerCfg.MaxParallelCommands; i++ {
			sshClient, err := DialSshWithPasswd(addr, routerCfg.User, routerCfg.Password)
			if err != nil {
				log.Fatal().Err(err).Msg("failed to establish SSH connection")
			}

			sshClients <- sshClient
		}
		l.Info().Msg("SSH connections established successfully")

		wg := sync.WaitGroup{}

		sshClient := <-sshClients
		currentRoutes := parseIpRoutes(sshClient.exec("show ip route"))
		l.Debug().Msgf("loaded %d static routes from router", len(currentRoutes))
		sshClients <- sshClient

		for _, interfaceCfg := range routerCfg.Interfaces {
			if interfaceCfg.Cleanup {
				removeRoutes := currentRoutes.filterIpRoutes(
					func(currentRoute IpRoute) bool {
						if currentRoute.InterfaceName != interfaceCfg.Name {
							return false
						}

						idx := slices.IndexFunc(
							provisionAddressesAll,
							func(provisionAddress IpRouteProvisionAddress) bool {
								return provisionAddress.Destination.Equals(currentRoute.Destination)
							},
						)
						return idx == -1
					},
				)

				if len(removeRoutes) > 0 {
					l.Info().Str("interface", interfaceCfg.Name).
						Msgf("remove %d outdated static routes", len(removeRoutes))

					bar := progressbar.Default(int64(len(removeRoutes)))
					wg.Add(len(removeRoutes))
					for _, route := range removeRoutes {
						sshClient = <-sshClients
						go func(route IpRoute, sshClient *SshClient) {
							defer func() {
								defer wg.Done()
								sshClients <- sshClient
								_ = bar.Add(1)
							}()

							sshClient.exec(
								fmt.Sprintf(
									"no ip route %s %s",
									route.Destination,
									route.InterfaceName,
								),
							)
						}(route, sshClient)
					}
					wg.Wait()
				} else {
					l.Info().Str("interface", interfaceCfg.Name).
						Msg("no outdated static routes")
				}
			}

			for _, route := range interfaceCfg.Routes {
				auto := ""
				if route.Auto {
					auto = "auto"
				}

				provisionAddresses, ok := provisionAddressesMap[route.Name]
				if !ok {
					l.Fatal().
						Str("name", route.Name).
						Msg("invalid name of static routes list")
				}

				provisionAddressesAdd := make(IpRouteProvisionList, 0)
				for _, provisionAddress := range provisionAddresses {
					idx := slices.IndexFunc(
						currentRoutes,
						func(currentRoute IpRoute) bool {
							return currentRoute.Destination.Equals(provisionAddress.Destination)
						},
					)
					if idx == -1 {
						provisionAddressesAdd = append(provisionAddressesAdd, provisionAddress)
					}
				}

				if len(provisionAddressesAdd) == 0 {
					l.Info().
						Str("interface", interfaceCfg.Name).
						Str("name", route.Name).
						Msgf("no static routes for adding")

					continue
				}

				l.Info().
					Str("interface", interfaceCfg.Name).
					Str("name", route.Name).
					Msgf("add %d static routes", len(provisionAddressesAdd))

				bar := progressbar.Default(int64(len(provisionAddressesAdd)))
				wg.Add(len(provisionAddressesAdd))
				for _, address := range provisionAddressesAdd {
					sshClient = <-sshClients
					go func(address IpRouteProvisionAddress, sshClient *SshClient) {
						defer func() {
							defer wg.Done()
							sshClients <- sshClient
							_ = bar.Add(1)
						}()

						sshClient.exec(
							// ip route 10.0.0.1 Wireguard0 auto !example
							fmt.Sprintf(
								"ip route %s %s %s !%s",
								address.Destination,
								interfaceCfg.Name,
								auto,
								address.Comment,
							),
						)
					}(address, sshClient)
				}
				wg.Wait()
			}
		}
	}
}

func flatAddresses(ipRouteAddressesMap map[string]IpRouteProvisionList) IpRouteProvisionList {
	out := make(IpRouteProvisionList, 0)
	for _, list := range ipRouteAddressesMap {
		out = append(out, list...)
	}
	return out
}

func parseStaticRoutesCfg(rootCfg RootConfig) map[string]IpRouteProvisionList {
	routesMap := make(map[string]IpRouteProvisionList)
	var list IpRouteProvisionList
	for name, routeCfgs := range rootCfg.StaticRoutes {
		l := log.With().Str("static-route-name", name).Logger()

		for _, routeCfg := range routeCfgs {
			targets := []string{routeCfg.Target}
			l.With().Str("target", routeCfg.Target).Logger()

			for _, resolverName := range routeCfg.Resolver {
				l.With().Str("resolver", resolverName).Logger()

				resolver := makeIpRouteAddressResolver(resolverName, l)
				var addresses []string
				for _, target := range targets {
					resolved, err := resolver(target)
					if err != nil {
						l.Fatal().Err(err).Msg("failed to resolve static route")
					}
					addresses = append(addresses, resolved...)
				}
				targets = addresses
			}

			comment := composeIpRouteAddressComment(routeCfg)
			items := make([]IpRouteProvisionAddress, len(targets))
			for i, target := range targets {
				items[i] = IpRouteProvisionAddress{
					Destination: IpRouteAddress(target),
					Comment:     comment,
				}
				l.Trace().Interface("resolved", items[i]).Msg("ip route target resolved")
			}
			list = append(list, items...)

			l.Debug().
				Str("target", routeCfg.Target).
				Int("len", len(items)).
				Msg("static route addresses resolved")
		}

		routesMap[name] = list
	}

	return routesMap
}

func composeIpRouteAddressComment(cfg StaticRouteConfig) string {
	comment := cfg.Target
	for i := 0; i < len(cfg.Resolver); i++ {
		comment = fmt.Sprintf("%s | %s()", comment, cfg.Resolver[i])
	}
	return comment
}

func parseArgs() map[string]interface{} {
	args, err := docopt.ParseArgs(usage, nil, version)
	if err != nil {
		panic(err)
	}

	return args
}
