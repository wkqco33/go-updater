// Package cli adapts the project's command surface to wcli while keeping
// command behavior easy to unit test.
package cli

import (
	"context"
	"errors"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/wkqco33/wcli"
)

type Command struct {
	Use, Short, Long string
	// Version enables wcli's --version flag when set.
	Version          string
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
	c.inner = &wcli.Command{Use: c.Use, Short: c.Short, Long: c.Long, Version: c.Version,
		OutWriter: c.out, ErrWriter: c.errOut, SilenceErrors: true}
	// Help renders with the full command path ("gu install [version]") while
	// keeping wcli's default template. wcli derives UsageLine from Use alone.
	c.inner.HelpFunc = func(wc *wcli.Command, w io.Writer) {
		savedUse, savedHelp, savedOut := wc.Use, wc.HelpFunc, wc.OutWriter
		wc.Use, wc.HelpFunc, wc.OutWriter = c.fullUse(), nil, w
		wc.Help()
		wc.Use, wc.HelpFunc, wc.OutWriter = savedUse, savedHelp, savedOut
	}
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
	return classify(c.inner.Execute(os.Args[1:]))
}
func (c *Command) ExecuteArgs(args []string) error {
	return c.ExecuteContext(context.Background(), args)
}
func (c *Command) ExecuteContext(parent context.Context, args []string) error {
	c.ensure()
	c.inner.ResetFlags()
	return classify(c.inner.ExecuteContext(parent, args))
}

// fullUse returns the command's path from the root, used by help output.
func (c *Command) fullUse() string {
	var parts []string
	for cur := c; cur != nil; cur = cur.parent {
		if use := strings.TrimSpace(cur.Use); use != "" {
			parts = append([]string{use}, parts...)
		}
	}
	return strings.Join(parts, " ")
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

// NoArgs rejects positional arguments, so a container command that only
// accepts subcommands reports an unknown command instead of silently running.
func NoArgs(_ *Command, args []string) error {
	if len(args) > 0 {
		return &ArgumentError{"unknown command " + strconv.Quote(args[0])}
	}
	return nil
}

type ArgumentError struct{ message string }

func (e *ArgumentError) Error() string { return e.message }

// UsageError marks an error caused by invalid invocation, such as an unknown
// flag, a wrong argument count, or an unknown command. Execute returns these
// wrapped so the process can separate usage failures from runtime failures.
type UsageError struct{ err error }

func NewUsageError(err error) *UsageError { return &UsageError{err: err} }

func (e *UsageError) Error() string { return e.err.Error() }

func (e *UsageError) Unwrap() error { return e.err }

// wcliUnknownCommand matches wcli's untagged error for an unknown command so
// the adapter can still classify it as a usage failure.
const wcliUnknownCommand = "unknown command "

func classify(err error) error {
	if err == nil {
		return nil
	}
	var argumentError *ArgumentError
	var flagError *wcli.FlagError
	var validationError *wcli.ValidationError
	if errors.As(err, &argumentError) || errors.As(err, &flagError) || errors.As(err, &validationError) {
		return NewUsageError(err)
	}
	if strings.HasPrefix(err.Error(), wcliUnknownCommand) {
		return NewUsageError(err)
	}
	return err
}

// ExitCode maps an Execute error to the documented process exit code: 0 for
// success, 2 for usage errors, and 1 for everything else.
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	var usageError *UsageError
	if errors.As(err, &usageError) {
		return 2
	}
	return 1
}

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
