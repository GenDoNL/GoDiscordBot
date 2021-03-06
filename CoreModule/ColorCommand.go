package CoreModule

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/g4s8/hexcolor"
	"github.com/gendonl/genbot/Bot"
	"image"
	"image/draw"
	"image/jpeg"
	"strconv"
	"strings"
)

func initColorCommand() (cc CoreCommand) {
	cc = CoreCommand{
		name:        "color",
		description: "Send the hex of the mentioned user, or the message author if no-one is mentioned." +
			"This command can also take a hex color starting with a # (e.g. #aabbcc).",
		usage:       "`%scolor [user/#hex]`",
		aliases:	 []string{"color", "colour", "clr"},
		permission:  discordgo.PermissionSendMessages,
		execute:     (*CoreModule).colorCommand,
	}
	return
}

func (c *CoreModule) colorCommand(cmd CoreCommand, s *discordgo.Session, m *discordgo.MessageCreate, data *Bot.ServerData) {
	input := strings.SplitN(m.Content, " ", 2)

	inputString := ""
	if len(input) > 1 {
		inputString = input[1]
	}

	var err error
	var target *discordgo.User
	var hex string
	if strings.HasPrefix(inputString, "#") {
		target = m.Author
		hex = inputString[1:]
	} else {
		target = c.Bot.GetCommandTarget(s, m, data, inputString)
		color := s.State.UserColor(target.ID, m.ChannelID)
		hex = fmt.Sprintf("%x", color)
	}

	complexResponse, err := c.handleHex(target, hex)
	if err != nil {
		_, err = s.ChannelMessageSend(m.ChannelID, "Unable to parse color")
		Log.Error(err)
		return
	}

	_, err = s.ChannelMessageSendComplex(m.ChannelID, complexResponse)

	if err != nil {
		Log.Error(err)
		return
	}
}

func (c *CoreModule) handleHex(author *discordgo.User, hex string) (response *discordgo.MessageSend, err error) {
	// Prepend 0s until the string is 6 long
	// Necessary since Discord removes prepended 0s
	for len(hex) < 6 {
		hex = "0" + hex
	}

	hex = "#" + hex

	color, err := hexcolor.Parse(hex)
	if err != nil {
		return
	}

	colorInt, err := strconv.ParseInt(hex[1:], 16, 32)
	if err != nil {
		return
	}

	// Generate image of that color
	img := image.NewRGBA(image.Rect(0, 0, 256, 64))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color}, image.Point{}, draw.Src)

	// Write image to reader
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	err = jpeg.Encode(w, img, &jpeg.Options{})
	if err != nil {
		return
	}

	// Create a discord file which takes the buffer.
	file := &discordgo.File{Name: "color.jpg", ContentType: "image/jpeg", Reader: bufio.NewReader(&b)}

	e := Bot.NewEmbed().
		SetAuthorFromUser(author).
		SetColor(int(colorInt)).
		SetTitle(hex).
		SetImage("attachment://color.jpg")

	response = &discordgo.MessageSend{
		Content: "",
		Embed:   e.MessageEmbed,
		Tts:     false,
		Files:   []*discordgo.File{file},
		File:    nil,
	}

	return
}