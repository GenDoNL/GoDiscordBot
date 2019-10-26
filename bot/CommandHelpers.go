package main

import (
	"errors"
	"github.com/bwmarrin/discordgo"
	"github.com/texttheater/golang-levenshtein/levenshtein"
	"math"
	"net/url"
	"strconv"
	"strings"
	"unicode"
)

// This function parse a discord.MessageCreate into a SentMessageData struct.
func parseMessage(m *discordgo.MessageCreate) SentMessageData {
	// Remove all white-space characters, except for new-lines.
	f := func(c rune) bool {
		return c != '\n' && unicode.IsSpace(c)
	}

	split := strings.FieldsFunc(m.Content, f)
	key := m.Content[:1]
	commandName := strings.ToLower(split[0][1:])
	if (len(commandName) > 1) && (commandName[len(commandName)-1] == '\n') {
		commandName = commandName[:len(commandName)-1]
	}

	content := split[1:]

	log.Error(content)

	return SentMessageData{key, commandName, content, m.ID, m.ChannelID, m.Mentions, m.Author}
}

// This function a string into an ID if the string is a mention.
func parseMention(str string) (string, error) {
	if len(str) < 5 || (string(str[0]) != "<" || string(str[1]) != "@" || string(str[len(str)-1]) != ">") {
		return "", errors.New("error while parsing mention, this is not an user")
	}

	res := str[2 : len(str)-1]

	// Necessary to allow nicknames.
	if string(res[0]) == "!" {
		res = res[1:]
	}

	return res, nil
}

// Returns the ServerData of a server, given a message object.
func getServerData(s *discordgo.Session, channelID string) *ServerData {
	channel, _ := s.Channel(channelID)

	servID := channel.GuildID

	if len(Servers) == 0 {
		Servers = make(map[string]*ServerData)
	}

	if serv, ok := Servers[servID]; ok {
		return serv
	}

	Servers[servID] = &ServerData{ID: servID, Key: "!"}
	return Servers[servID]

}

// Checks whether a user id (String) is in a slice of users.
func userInSlice(a string, list []*discordgo.User) bool {
	for _, b := range list {
		if b.ID == a {
			return true
		}
	}
	return false
}

// Gets the specific role by name out of a role list.
func getRoleByName(name string, roles []*discordgo.Role) (r discordgo.Role, e error) {
	for _, elem := range roles {
		if elem.Name == name {
			r = *elem
			return
		}
	}
	e = errors.New("Role name not found in the specified role array: " + name)
	return
}

// Gets the permission override object from a role id.
func getRolePermissions(id string, perms []*discordgo.PermissionOverwrite) (p discordgo.PermissionOverwrite, e error) {
	for _, elem := range perms {
		if elem.ID == id {
			p = *elem
			return
		}
	}
	e = errors.New("permissions not found in the specified role: " + id)
	return
}

func getRolePermissionsByName(ch *discordgo.Channel, sv *discordgo.Guild, name string) (p discordgo.PermissionOverwrite, e error) {
	//get role object for given name
	role, _ := getRoleByName(name, sv.Roles)
	return getRolePermissions(role.ID, ch.PermissionOverwrites)
}

func getRoleById(s *discordgo.Session, data *ServerData, id string) (*discordgo.Role, error) {
	g, _ := s.Guild(data.ID)
	for _, role := range g.Roles {
		if role.ID == id {
			println(role.Name)
			return role, nil
		}
	}
	return nil, errors.New("role not found in list")
}

// isValidUrl tests a string to determine if it is a url or not.
func isValidUrl(toTest string) bool {
	_, err := url.ParseRequestURI(toTest)
	if err != nil {
		return false
	} else {
		return true
	}
}

// Creates a command in the given server given a name and a message.
func createCommand(data *ServerData, commandName, message string) error {
	name := strings.ToLower(commandName)
	if strings.Contains(name, "\n") {
		log.Info("Trying to add command name with newline, aborted.")
		return errors.New("trying to add command with a name that contains a new line")
	}
	data.CustomCommands[name] = &CommandData{name, message}
	writeServerData()
	return nil
}

func checkCommandsMap(data *ServerData) {
	if len(data.CustomCommands) == 0 {
		data.CustomCommands = make(map[string]*CommandData)
	}
}

func checkChannelsMap(data *ServerData) {
	if len(data.Channels) == 0 {
		data.Channels = make(map[string]*ChannelData)
	}
}

func findLastMessageWithAttachOrEmbed(s *discordgo.Session, msg SentMessageData, amount int) (result string, e error) {
	msgList, _ := s.ChannelMessages(msg.ChannelID, amount, msg.MessageID, "", "")

	for _, x := range msgList {
		if len(x.Embeds) > 0 {
			result = x.Embeds[0].URL
			e = nil
			return
		} else if len(x.Attachments) > 0 {
			result = x.Attachments[0].URL
			e = nil
			return
		}
	}

	result = ""
	e = errors.New("Unable to find message with attachment or embed")
	return
}

func getAccountCreationDate(user *discordgo.User) (timestamp int64) {
	id, _ := strconv.ParseUint(user.ID, 10, 64)
	timestamp = int64(((id >> 22) + 1420070400000) / 1000) // Divided by 1000 since we want seconds rather than ms
	return
}

func getClosestUserByName(s *discordgo.Session, data *ServerData, user string) (foundUser *discordgo.User, err error) {
	currentMaxDistance := math.MaxInt64

	guild, err := s.Guild(data.ID)

	for _, nick := range guild.Members {
		userName := nick.User.Username

		// Prefer Server nickname over Discord username
		if nick.Nick != "" {
			userName = nick.Nick
		}

		levenDistance := levenshtein.DistanceForStrings([]rune(userName), []rune(user), levenshtein.DefaultOptions)
		if levenDistance < currentMaxDistance {
			currentMaxDistance = levenDistance
			foundUser = nick.User
		}
	}

	return
}

func getCommandTarget(s *discordgo.Session, msg SentMessageData, data *ServerData) (target *discordgo.User) {
	if len(msg.Mentions) > 0 {
		target = msg.Mentions[0]
	} else {
		if len(msg.Content) > 0 {
			trg, err := getClosestUserByName(s, data, strings.Join(msg.Content, " "))
			target = trg
			if err != nil {
				target = msg.Author // Fallback if error occurs
			}
		} else {
			target = msg.Author
		}
	}
	return
}
