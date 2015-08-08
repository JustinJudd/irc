package irc

import (
	"fmt"
	"sync"
)

// UserMode  - RFC 1459 Section 4.2.3.2 and RFC 2812 Section 3.1.5
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

// UserModes contains the supported User UserMode types
var UserModes = map[UserMode]interface{}{
	UserModeAway:          nil,
	UserModeInvisible:     nil,
	UserModeWallOps:       nil,
	UserModeRestricted:    nil,
	UserModeOperator:      nil,
	UserModeLocalOperator: nil,
	UserModeServerNotice:  nil,
}

// UserModeSet provides means for storing and checking UserModes
type UserModeSet struct {
	userModes map[UserMode]interface{}
	mutex     sync.RWMutex
}

// NewUserModeSet creates and returns a new UserModeSet
func NewUserModeSet() *UserModeSet {
	u := UserModeSet{}
	u.userModes = map[UserMode]interface{}{}
	return &u
}

// AddMode adds a mode to the UserModeSet
func (u *UserModeSet) AddMode(mode UserMode) {
	u.mutex.Lock()
	defer u.mutex.Unlock()
	u.userModes[mode] = nil

}

// RemoveMode removes a mode from the UserModeSet
func (u *UserModeSet) RemoveMode(mode UserMode) {
	u.mutex.Lock()
	defer u.mutex.Unlock()
	delete(u.userModes, mode)

}

// HasMode detects if a UserMode is contained in the UserModeSet
func (u *UserModeSet) HasMode(mode UserMode) bool {
	u.mutex.RLock()
	defer u.mutex.RUnlock()
	_, found := u.userModes[mode]
	return found
}

// String formats the UserModeString to be returned for MODE queries
func (u *UserModeSet) String() string {
	s := ""
	if len(u.userModes) > 0 {
		s = "+"
	}
	for m := range u.userModes {
		s += string(m)
	}

	return s
}

// ModeModifier - RFC 1459 Section 4.2.3 and RFC 2812 Section 3.1.5
type ModeModifier rune

const (
	ModeModifierAdd    ModeModifier = '+'
	ModeModifierRemove ModeModifier = '-'
)

// ChannelMode -  RFC 1459 Section 4.2.3.1 and RFC 2812 Section 3.2.3 and RFC 2811 Section 4
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

// ChannelModeSet represents a set of active ChannelModes
type ChannelModeSet struct {
	modes map[ChannelMode]interface{}
	mutex sync.RWMutex
}

// NewChannelModeSet creates and returns a new ChannelModeSet
func NewChannelModeSet() *ChannelModeSet {
	c := ChannelModeSet{}
	c.modes = map[ChannelMode]interface{}{}
	return &c
}

// AddMode adds a ChannelMode as active
func (c *ChannelModeSet) AddMode(mode ChannelMode) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.modes[mode] = nil

}

// AddModeWithValue adds a ChannelMode as active with a value
func (c *Channel) AddModeWithValue(mode ChannelMode, value interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.modes[mode] = value
}

// RemoveMode removes the given mode from the active set
func (c *ChannelModeSet) RemoveMode(mode ChannelMode) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.modes, mode)

}

// HasMode determines if the ChannelModeSet contains the given ChannelMode
func (c *ChannelModeSet) HasMode(mode ChannelMode) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	_, found := c.modes[mode]
	return found

}

// GetMode determines if the ChannelModeSet contains the given ChannelMode
func (c *ChannelModeSet) GetMode(mode ChannelMode) interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.modes[mode]

}

// String returns the ChannelModeSet formatted for the MODE queries
func (c *ChannelModeSet) String() string {
	s := "+"

	params := []interface{}{}
	for m, param := range c.modes {
		switch m {
		case ChannelModeKey, ChannelModeLimit, ChannelModeModerated, ChannelModeAnonymous, ChannelModeInviteOnly, ChannelModePrivate, ChannelModeSecret, ChannelModeTopic:
		default:
			continue
		}
		s += string(m)
		params = append(params, param)
	}
	for _, param := range params {
		if param != nil {
			s += fmt.Sprintf(" %v", param)
		}

	}

	return s
}
