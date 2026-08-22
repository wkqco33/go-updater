// Package cli adapts the project's command surface to wcli while keeping
// command behavior easy to unit test.
package cli

import (
	"context"
	"io"
	"os"
	"strconv"

	"github.com/wkqco33/wcli"
)

type Context struct {
	inner *wcli.Context
	cmd   *Command
}

func (c *Context) Context() context.Context { return c.inner.Context }
func (c *Context) Args() []string           { return c.inner.Args }

type Command struct {
	Use, Short, Long string
	Args             func(*Command, []string) error
	RunE             func(*Command, []string) error
	PersistentPreRun func(*Command, []string)

	inner       *wcli.Command
	parent      *Command
	context     context.Context
	in          io.Reader
	out, errOut io.Writer
	flags       *FlagSet
	persistent  *FlagSet
}

func (c *Command) ensure() {
	if c.inner != nil {
		return
	}
	c.inner = &wcli.Command{Use: c.Use, Short: c.Short, Long: c.Long,
		OutWriter: c.out, ErrWriter: c.errOut, SilenceErrors: true}
	c.inner.Run = func(ctx *wcli.Context) error {
		c.context = ctx.Context
		c.syncInt64Flags()
		args := ctx.Args
		if c.Args != nil {
			if err := c.Args(c, args); err != nil {
				return err
			}
		}
		if c.RunE == nil {
			return nil
		}
		return c.RunE(c, args)
	}
	if c.PersistentPreRun != nil {
		c.inner.PersistentPreRun = func(ctx *wcli.Context) error {
			c.PersistentPreRun(c, ctx.Args)
			return nil
		}
	}
}

func (c *Command) AddCommand(children ...*Command) {
	c.ensure()
	for _, child := range children {
		child.parent = c
		child.ensure()
		c.inner.AddCommand(child.inner)
	}
}

func (c *Command) Execute() error {
	c.ensure()
	return c.inner.Execute(os.Args[1:])
}
func (c *Command) ExecuteArgs(args []string) error {
	return c.ExecuteContext(context.Background(), args)
}
func (c *Command) ExecuteContext(parent context.Context, args []string) error {
	c.ensure()
	c.inner.ResetFlags()
	return c.inner.ExecuteContext(parent, args)
}
func (c *Command) Context() context.Context {
	if c.context != nil {
		return c.context
	}
	return context.Background()
}
func (c *Command) OutOrStdout() io.Writer {
	if c.out != nil {
		return c.out
	}
	if c.parent != nil {
		return c.parent.OutOrStdout()
	}
	return os.Stdout
}
func (c *Command) ErrOrStderr() io.Writer {
	if c.errOut != nil {
		return c.errOut
	}
	if c.parent != nil {
		return c.parent.ErrOrStderr()
	}
	return os.Stderr
}
func (c *Command) InOrStdin() io.Reader {
	if c.in != nil {
		return c.in
	}
	if c.parent != nil {
		return c.parent.InOrStdin()
	}
	return os.Stdin
}
func (c *Command) SetOut(w io.Writer) {
	c.out = w
	if c.inner != nil {
		c.inner.OutWriter = w
	}
}
func (c *Command) SetErr(w io.Writer) {
	c.errOut = w
	if c.inner != nil {
		c.inner.ErrWriter = w
	}
}
func (c *Command) SetIn(r io.Reader) { c.in = r }
func (c *Command) Help()             { c.ensure(); c.inner.Help() }

func MaximumNArgs(n int) func(*Command, []string) error {
	return func(_ *Command, args []string) error {
		if len(args) > n {
			return &ArgumentError{"accepts at most " + strconv.Itoa(n) + " arg(s)"}
		}
		return nil
	}
}
func MinimumNArgs(n int) func(*Command, []string) error {
	return func(_ *Command, args []string) error {
		if len(args) < n {
			return &ArgumentError{"requires at least " + strconv.Itoa(n) + " arg(s)"}
		}
		return nil
	}
}
func ExactArgs(n int) func(*Command, []string) error {
	return func(_ *Command, args []string) error {
		if len(args) != n {
			return &ArgumentError{"accepts " + strconv.Itoa(n) + " arg(s), received " + strconv.Itoa(len(args))}
		}
		return nil
	}
}

type ArgumentError struct{ message string }

func (e *ArgumentError) Error() string { return e.message }

type FlagSet struct {
	inner       *wcli.FlagSet
	int64Values []func()
}

func (c *Command) Flags() *FlagSet {
	c.ensure()
	if c.flags == nil {
		c.flags = &FlagSet{inner: c.inner.Flags()}
	}
	return c.flags
}
func (c *Command) PersistentFlags() *FlagSet {
	c.ensure()
	if c.persistent == nil {
		c.persistent = &FlagSet{inner: c.inner.PersistentFlags()}
	}
	return c.persistent
}
func (f *FlagSet) StringVar(p *string, name, shorthand, value, usage string) {
	f.inner.StringVar(p, name, shorthand, value, usage)
}
func (f *FlagSet) IntVar(p *int, name, shorthand string, value int, usage string) {
	f.inner.IntVar(p, name, shorthand, value, usage)
}
func (f *FlagSet) Int64Var(p *int64, name, shorthand string, value int64, usage string) {
	shadow := int(value)
	f.inner.IntVar(&shadow, name, shorthand, int(value), usage)
	f.int64Values = append(f.int64Values, func() { *p = int64(shadow) })
}
func (f *FlagSet) BoolVar(p *bool, name, shorthand string, value bool, usage string) {
	f.inner.BoolVar(p, name, shorthand, value, usage)
}
func (f *FlagSet) StringVarP(p *string, name, shorthand, value, usage string) {
	f.StringVar(p, name, shorthand, value, usage)
}
func (f *FlagSet) Changed(name string) bool { return f.inner.Changed(name) }
func (f *FlagSet) syncInt64Flags() {
	for _, sync := range f.int64Values {
		sync()
	}
}
func (c *Command) syncInt64Flags() {
	if c.flags != nil {
		c.flags.syncInt64Flags()
	}
	if c.persistent != nil {
		c.persistent.syncInt64Flags()
	}
}
