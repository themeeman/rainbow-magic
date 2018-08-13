package main

import (
	"github.com/bwmarrin/discordgo"
	"flag"
	"log"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"strings"
	"time"
)

var token string

type Rainbow struct {
	GuildID string
	RoleID  string
	Quit    chan struct{}
}

const (
	Red    = 0xff0000
	Orange = 0xff8c00
	Yellow = 0xffff00
	Green  = 0x009900
	Blue   = 0x000099
	Indigo = 0x4b0082
	Violet = 0xff00ff
)

var rainbows []*Rainbow
var colors = []int{
	Red,
	Orange,
	Yellow,
	Green,
	Blue,
	Indigo,
	Violet,
}

func contains(rainbows []*Rainbow, role string) (int, bool) {
	for i, v := range rainbows {
		if v.RoleID == role {
			return i, true
		}
	}
	return 0, false
}

func hasPermission(session *discordgo.Session, guildID string, userID string, permission int) (bool, error) {
	member, err := session.GuildMember(guildID, userID)
	if err != nil {
		return false, err
	}
	for _, id := range member.Roles {
		role, err := sessionGuildRole(session, guildID, id)
		if err != nil {
			return false, err
		}
		if role.Permissions & permission == permission {
			return true, nil
		}
	}
	return false, nil
}

func sessionGuildRole(session *discordgo.Session, guildID string, roleID string) (*discordgo.Role, error) {
	roles, err := session.GuildRoles(guildID)
	if err != nil {
		return nil, err
	}
	for _, r := range roles {
		if r.ID == roleID {
			return r, nil
		}
	}
	return nil, fmt.Errorf("role not found %s", roleID)
}

func ready(session *discordgo.Session, _ *discordgo.Ready) {
	session.UpdateStatus(0, "rainbow <@Role>")
}

func messageCreate(session *discordgo.Session, message *discordgo.MessageCreate) {
	if message.Author.Bot {
		return
	}
	fields := strings.Fields(strings.ToLower(message.Content))
	if len(fields) == 2 && fields[0] == "rainbow" && len(message.MentionRoles) == 1 {
		ch, err := session.Channel(message.ChannelID)
		if err != nil {
			return
		}
		if ok, err := hasPermission(session, ch.GuildID, message.Author.ID, discordgo.PermissionAdministrator); !ok || err != nil {
			session.ChannelMessageSend(message.ChannelID, "Insufficient permissions - You need Administrator!")
			return
		}
		role := message.MentionRoles[0]
		if index, ok := contains(rainbows, role); ok {
			fmt.Println(index, ok)
			r := rainbows[index]
			if index == len(rainbows)-1 {
				rainbows = rainbows[:index]
			} else {
				rainbows = append(rainbows[:index], rainbows[index+1:]...)
			}
			if r.Quit != nil {
				session.ChannelMessageSend(message.ChannelID, "Successfully ended rainbow")
				r.Quit <- struct{}{}
			}
		} else {
			role, err := sessionGuildRole(session, ch.GuildID, role)
			if err != nil {
				session.ChannelMessageSend(ch.ID, "Invalid Role!")
			}
			q := make(chan struct{})
			r := Rainbow{
				GuildID: ch.GuildID,
				RoleID:  role.ID,
				Quit:    q,
			}
			rainbows = append(rainbows, &r)
			session.ChannelMessageSend(message.ChannelID, "Successfully started Rainbow")
			go func() {
				var i int
				for {
					select {
					case <-r.Quit:
						return
					default:
						role, err = sessionGuildRole(session, r.GuildID, r.RoleID)
						if err != nil {
							log.Println(err)
							continue
						}
						_, err = session.GuildRoleEdit(r.GuildID, r.RoleID, role.Name, colors[i], role.Hoist, role.Permissions, role.Mentionable)
						if err != nil {
							log.Println(err)
							continue
						}
						i += 1
						if i >= len(colors) {
							i %= len(colors)
						}
						time.Sleep(time.Second)
					}
				}
			}()
		}
	}
}

func init() {
	flag.StringVar(&token, "t", "", "Token")
	flag.Parse()
}

func main() {
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatalln(err)
	}
	dg.AddHandler(messageCreate)
	dg.AddHandler(ready)
	err = dg.Open()
	if err != nil {
		log.Fatalln(err)
	}
	defer dg.Close()
	fmt.Println("Rainbow Magic is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
}

