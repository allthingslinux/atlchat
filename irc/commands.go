package irc

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

type Command interface {
	Client() *Client
	Source() Identifier
	Reply(Reply)
	HandleServer(*Server)
}

type EditableCommand interface {
	Command
	SetBase(*Client)
}

var (
	NotEnoughArgsError = errors.New("not enough arguments")
	ErrParseCommand    = errors.New("failed to parse message")
	parseCommandFuncs  = map[string]func([]string) (EditableCommand, error){
		"JOIN":    NewJoinCommand,
		"MODE":    NewModeCommand,
		"NICK":    NewNickCommand,
		"PART":    NewPartCommand,
		"PASS":    NewPassCommand,
		"PING":    NewPingCommand,
		"PONG":    NewPongCommand,
		"PRIVMSG": NewPrivMsgCommand,
		"QUIT":    NewQuitCommand,
		"TOPIC":   NewTopicCommand,
		"USER":    NewUserMsgCommand,
		"WHO":     NewWhoCommand,
		"WHOIS":   NewWhoisCommand,
	}
)

type BaseCommand struct {
	client *Client
}

func (command *BaseCommand) Client() *Client {
	return command.client
}

func (command *BaseCommand) SetBase(c *Client) {
	*command = BaseCommand{c}
}

func (command *BaseCommand) Source() Identifier {
	return command.client
}

func (command *BaseCommand) Reply(reply Reply) {
	command.client.Replies() <- reply
}

func ParseCommand(line string) (EditableCommand, error) {
	command, args := parseLine(line)
	constructor := parseCommandFuncs[command]
	if constructor == nil {
		return NewUnknownCommand(command, args), nil
	}
	return constructor(args)
}

func parseArg(line string) (arg string, rest string) {
	if line == "" {
		return
	}

	if strings.HasPrefix(line, ":") {
		arg = line[1:]
	} else {
		parts := strings.SplitN(line, " ", 2)
		arg = parts[0]
		if len(parts) > 1 {
			rest = parts[1]
		}
	}
	return
}

func parseLine(line string) (command string, args []string) {
	args = make([]string, 0)
	for arg, rest := parseArg(line); arg != ""; arg, rest = parseArg(rest) {
		args = append(args, arg)
	}
	command, args = strings.ToUpper(args[0]), args[1:]
	return
}

// <command> [args...]

type UnknownCommand struct {
	BaseCommand
	command string
	args    []string
}

func (cmd *UnknownCommand) String() string {
	return fmt.Sprintf("UNKNOWN(command=%s, args=%s)", cmd.command, cmd.args)
}

func NewUnknownCommand(command string, args []string) *UnknownCommand {
	return &UnknownCommand{
		command: command,
		args:    args,
	}
}

// PING <server1> [ <server2> ]

type PingCommand struct {
	BaseCommand
	server  string
	server2 string
}

func (cmd *PingCommand) String() string {
	return fmt.Sprintf("PING(server=%s, server2=%s)", cmd.server, cmd.server2)
}

func NewPingCommand(args []string) (EditableCommand, error) {
	if len(args) < 1 {
		return nil, NotEnoughArgsError
	}
	msg := &PingCommand{
		server: args[0],
	}
	if len(args) > 1 {
		msg.server2 = args[1]
	}
	return msg, nil
}

// PONG <server> [ <server2> ]

type PongCommand struct {
	BaseCommand
	server1 string
	server2 string
}

func (cmd *PongCommand) String() string {
	return fmt.Sprintf("PONG(server1=%s, server2=%s)", cmd.server1, cmd.server2)
}

func NewPongCommand(args []string) (EditableCommand, error) {
	if len(args) < 1 {
		return nil, NotEnoughArgsError
	}
	message := &PongCommand{
		server1: args[0],
	}
	if len(args) > 1 {
		message.server2 = args[1]
	}
	return message, nil
}

// PASS <password>

type PassCommand struct {
	BaseCommand
	password string
}

func (cmd *PassCommand) String() string {
	return fmt.Sprintf("PASS(password=%s)", cmd.password)
}

func NewPassCommand(args []string) (EditableCommand, error) {
	if len(args) < 1 {
		return nil, NotEnoughArgsError
	}
	return &PassCommand{
		password: args[0],
	}, nil
}

// NICK <nickname>

type NickCommand struct {
	BaseCommand
	nickname string
}

func (m *NickCommand) String() string {
	return fmt.Sprintf("NICK(nickname=%s)", m.nickname)
}

func NewNickCommand(args []string) (EditableCommand, error) {
	if len(args) != 1 {
		return nil, NotEnoughArgsError
	}
	return &NickCommand{
		nickname: args[0],
	}, nil
}

// USER <user> <mode> <unused> <realname>

type UserMsgCommand struct {
	BaseCommand
	user     string
	mode     uint8
	unused   string
	realname string
}

func (cmd *UserMsgCommand) String() string {
	return fmt.Sprintf("USER(user=%s, mode=%o, unused=%s, realname=%s)",
		cmd.user, cmd.mode, cmd.unused, cmd.realname)
}

func NewUserMsgCommand(args []string) (EditableCommand, error) {
	if len(args) != 4 {
		return nil, NotEnoughArgsError
	}
	msg := &UserMsgCommand{
		user:     args[0],
		unused:   args[2],
		realname: args[3],
	}
	mode, err := strconv.ParseUint(args[1], 10, 8)
	if err == nil {
		msg.mode = uint8(mode)
	}
	return msg, nil
}

// QUIT [ <Quit Command> ]

type QuitCommand struct {
	BaseCommand
	message string
}

func (cmd *QuitCommand) String() string {
	return fmt.Sprintf("QUIT(message=%s)", cmd.message)
}

func NewQuitCommand(args []string) (EditableCommand, error) {
	msg := &QuitCommand{}
	if len(args) > 0 {
		msg.message = args[0]
	}
	return msg, nil
}

// JOIN ( <channel> *( "," <channel> ) [ <key> *( "," <key> ) ] ) / "0"

type JoinCommand struct {
	BaseCommand
	channels map[string]string
	zero     bool
}

func (cmd *JoinCommand) String() string {
	return fmt.Sprintf("JOIN(channels=%s, zero=%t)", cmd.channels, cmd.zero)
}

func NewJoinCommand(args []string) (EditableCommand, error) {
	msg := &JoinCommand{
		channels: make(map[string]string),
	}

	if len(args) == 0 {
		return nil, NotEnoughArgsError
	}

	if args[0] == "0" {
		msg.zero = true
		return msg, nil
	}

	channels := strings.Split(args[0], ",")
	keys := make([]string, len(channels))
	if len(args) > 1 {
		for i, key := range strings.Split(args[1], ",") {
			keys[i] = key
		}
	}
	for i, channel := range channels {
		msg.channels[channel] = keys[i]
	}

	return msg, nil
}

// PART <channel> *( "," <channel> ) [ <Part Command> ]

type PartCommand struct {
	BaseCommand
	channels []string
	message  string
}

func (cmd *PartCommand) Message() string {
	if cmd.message == "" {
		return cmd.Source().Nick()
	}
	return cmd.message
}

func (cmd *PartCommand) String() string {
	return fmt.Sprintf("PART(channels=%s, message=%s)", cmd.channels, cmd.message)
}

func NewPartCommand(args []string) (EditableCommand, error) {
	if len(args) < 1 {
		return nil, NotEnoughArgsError
	}
	msg := &PartCommand{
		channels: strings.Split(args[0], ","),
	}
	if len(args) > 1 {
		msg.message = args[1]
	}
	return msg, nil
}

// PRIVMSG <target> <message>

type PrivMsgCommand struct {
	BaseCommand
	target  string
	message string
}

func (cmd *PrivMsgCommand) String() string {
	return fmt.Sprintf("PRIVMSG(target=%s, message=%s)", cmd.target, cmd.message)
}

func NewPrivMsgCommand(args []string) (EditableCommand, error) {
	if len(args) < 2 {
		return nil, NotEnoughArgsError
	}
	return &PrivMsgCommand{
		target:  args[0],
		message: args[1],
	}, nil
}

func (m *PrivMsgCommand) TargetIsChannel() bool {
	return IsChannel(m.target)
}

// TOPIC [newtopic]

type TopicCommand struct {
	BaseCommand
	channel string
	topic   string
}

func (cmd *TopicCommand) String() string {
	return fmt.Sprintf("TOPIC(channel=%s, topic=%s)", cmd.channel, cmd.topic)
}

func NewTopicCommand(args []string) (EditableCommand, error) {
	if len(args) < 1 {
		return nil, NotEnoughArgsError
	}
	msg := &TopicCommand{
		channel: args[0],
	}
	if len(args) > 1 {
		msg.topic = args[1]
	}
	return msg, nil
}

type Mode rune

const (
	Away          Mode = 'a'
	Invisible     Mode = 'i'
	WallOps       Mode = 'w'
	Restricted    Mode = 'r'
	Operator      Mode = 'o'
	LocalOperator Mode = 'O'
	ServerNotice  Mode = 's'
)

type ModeChange struct {
	mode Mode
	add  bool // false => remove
}

func (change *ModeChange) String() string {
	sig := "+"
	if !change.add {
		sig = "-"
	}
	return fmt.Sprintf("%s%s", sig, change.mode)
}

type ModeCommand struct {
	BaseCommand
	nickname string
	changes  []ModeChange
}

func (cmd *ModeCommand) String() string {
	return fmt.Sprintf("MODE(nickname=%s, changes=%s)", cmd.nickname, cmd.changes)
}

func stringToRunes(str string) <-chan rune {
	runes := make(chan rune)
	go func() {
		for len(str) > 0 {
			rune, size := utf8.DecodeRuneInString(str)
			runes <- rune
			str = str[size:]
		}
		close(runes)
	}()
	return runes
}

type ChannelModeCommand struct {
	BaseCommand
	channel string
}

// MODE <channel> *( ( "-" / "+" ) *<modes> *<modeparams> )
func NewChannelModeCommand(args []string) (EditableCommand, error) {
	cmd := &ChannelModeCommand{
		channel: args[0],
	}
	// TODO implement channel mode changes
	return cmd, nil
}

// MODE <nickname> *( ( "+" / "-" ) *( "i" / "w" / "o" / "O" / "r" ) )
func NewUserModeCommand(args []string) (EditableCommand, error) {
	cmd := &ModeCommand{
		nickname: args[0],
		changes: make([]ModeChange,
			utf8.RuneCountInString(strings.Join(args[1:], ""))-len(args[1:])),
	}

	index := 0
	for _, arg := range args[1:] {
		modeChange := stringToRunes(arg)
		sig := <-modeChange
		if sig != '+' && sig != '-' {
			return nil, ErrParseCommand
		}

		add := sig == '+'
		for mode := range modeChange {
			cmd.changes[index] = ModeChange{
				mode: Mode(mode),
				add:  add,
			}
			index += 1
		}
	}

	return cmd, nil
}

func NewModeCommand(args []string) (EditableCommand, error) {
	if len(args) == 0 {
		return nil, NotEnoughArgsError
	}

	if IsChannel(args[0]) {
		return NewChannelModeCommand(args)
	} else {
		return NewUserModeCommand(args)
	}
}

type WhoisCommand struct {
	BaseCommand
	target string
	masks  []string
}

// WHOIS [ <target> ] <mask> *( "," <mask> )
func NewWhoisCommand(args []string) (EditableCommand, error) {
	if len(args) < 1 {
		return nil, NotEnoughArgsError
	}

	var masks string
	var target string

	if len(args) > 1 {
		target = args[0]
		masks = args[1]
	} else {
		masks = args[0]
	}

	return &WhoisCommand{
		target: target,
		masks:  strings.Split(masks, ","),
	}, nil
}

type WhoCommand struct {
	BaseCommand
	mask         string
	operatorOnly bool
}

// WHO [ <mask> [ "o" ] ]
func NewWhoCommand(args []string) (EditableCommand, error) {
	cmd := &WhoCommand{}

	if len(args) > 0 {
		cmd.mask = args[0]
	}

	if (len(args) > 1) && (args[1] == "o") {
		cmd.operatorOnly = true
	}

	return cmd, nil
}
