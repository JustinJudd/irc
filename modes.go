package irc

import "sync"

// UserMode  - RFC 1459 4.2.3.2 and RFC 2812 3.1.5
type UserMode rune

const (
	UserModeAway          UserMode = 'a'
	UserModeInvisible     UserMode = 'i'
	UserModeWallOps       UserMode = 'w'
	UserModeRestricted    UserMode = 'r'
	UserModeOperator      UserMode = 'o'
	UserModeLocalOperator UserMode = 'O'
	UserModeServerNotice  UserMode = 's' //obsolete
)

var UserModes = map[UserMode]interface{}{
	UserModeAway:          nil,
	UserModeInvisible:     nil,
	UserModeWallOps:       nil,
	UserModeRestricted:    nil,
	UserModeOperator:      nil,
	UserModeLocalOperator: nil,
	UserModeServerNotice:  nil,
}

type UserModeSet struct {
	userModes map[UserMode]interface{}
	mutex     sync.Mutex
}

func NewUserModeSet() UserModeSet {
	u := UserModeSet{}
	u.userModes = map[UserMode]interface{}{}
	return u
}

func (u *UserModeSet) AddMode(mode UserMode) {
	u.mutex.Lock()
	defer u.mutex.Unlock()
	u.userModes[mode] = nil

}

func (u *UserModeSet) RemoveMode(mode UserMode) {
	u.mutex.Lock()
	defer u.mutex.Unlock()
	delete(u.userModes, mode)

}

func (u *UserModeSet) String() string {
	s := ""
	for m, _ := range u.userModes {
		s += string(m)
	}

	return s
}

// ModeModifier - RFC 1459 4.2.3 and RFC 2812 3.1.5
type ModeModifier rune

const (
	ModeModifierAdd    ModeModifier = '+'
	ModeModifierRemove ModeModifier = '-'
)

// ChannelMode -  RFC 1459 4.2.3.1 and RFC 2812 3.2.3 and RFC 2811 4
type ChannelMode rune

const (
	ChannelModeCreator  ChannelMode = 'O'
	ChannelModeOperator ChannelMode = 'o'
	ChannelModeVoice    ChannelMode = 'v'

	ChannelModeAnonymous         ChannelMode = 'a'
	ChannelModeInviteOnly        ChannelMode = 'i'
	ChannelModeModerated         ChannelMode = 'm'
	ChannelModeNoOutsideMessages ChannelMode = 'n'
	ChannelModeQuiet             ChannelMode = 'q'
	ChannelModePrivate           ChannelMode = 'p'
	ChannelModeSecret            ChannelMode = 's'
	ChannelModeReOp              ChannelMode = 'r'
	ChannelModeTopic             ChannelMode = 't'

	ChannelModeKey   ChannelMode = 'k'
	ChannelModeLimit ChannelMode = 'l'

	ChannelModeBan            ChannelMode = 'b'
	ChannelModeExceptionMask  ChannelMode = 'e'
	ChannelModeInvitationMask ChannelMode = 'I'
)

type ChannelModeSet struct {
	Modes map[ChannelMode]interface{}
	mutex sync.Mutex
}

func NewChannelModeSet() ChannelModeSet {
	c := ChannelModeSet{}
	c.Modes = map[ChannelMode]interface{}{}
	return c
}

func (c *ChannelModeSet) AddMode(mode ChannelMode) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.Modes[mode] = nil

}

func (c *ChannelModeSet) RemoveMode(mode ChannelMode) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.Modes, mode)

}
