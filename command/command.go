package command

import "flag"

type arrayFlags []string

func (i *arrayFlags) String() string {
	groups := ""
	for _, group := range *i {
		groups += group
	}
	return groups
}

func (i *arrayFlags) Set(value string) error {
	// Don't add duplicates
	for _, b := range *i {
		if b == value {
			return nil
		}
	}

	*i = append(*i, value)
	return nil
}

type Command struct {
	Name         string
	Command      *flag.FlagSet
	strings      map[string]*string
	stringArrays map[string]*arrayFlags
	bools        map[string]*bool
}

func New(name string) *Command {
	return &Command{
		Name:         name,
		Command:      flag.NewFlagSet(name, flag.ExitOnError),
		strings:      make(map[string]*string),
		bools:        make(map[string]*bool),
		stringArrays: make(map[string]*arrayFlags),
	}
}

func (c *Command) AddString(name string, defaultValue string, usage string) {
	c.strings[name] = c.Command.String(name, defaultValue, usage)
}

func (c *Command) GetString(name string) string {
	if value, ok := c.strings[name]; ok {
		return *value
	}
	return ""
}

func (c *Command) AddStringArray(name string, defaultValue arrayFlags, usage string) {
	c.Command.Var(&defaultValue, name, usage)
	c.stringArrays[name] = &defaultValue
}

func (c *Command) GetStringArray(name string) []string {
	if value, ok := c.stringArrays[name]; ok {
		return *value
	}
	return []string{}
}

func (c *Command) AddBool(name string, defaultValue bool, usage string) {
	c.bools[name] = c.Command.Bool(name, defaultValue, usage)
}

func (c *Command) GetBool(name string) bool {
	if value, ok := c.bools[name]; ok {
		return *value
	}
	return false
}
