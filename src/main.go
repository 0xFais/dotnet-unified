package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type Source struct{ Name, Root string }
type State struct {
	Links map[string]string `json:"links"`
}

func exists(p string) bool { _, e := os.Stat(p); return e == nil }
func dirs(p string) []string {
	xs, e := os.ReadDir(p)
	if e != nil {
		return nil
	}
	var a []string
	for _, x := range xs {
		if x.IsDir() {
			a = append(a, x.Name())
		}
	}
	sort.Strings(a)
	return a
}
func scoop(exe string) string {
	if s := os.Getenv("SCOOP"); s != "" {
		return s
	}
	p := filepath.Dir(exe)
	for {
		if strings.EqualFold(filepath.Base(filepath.Dir(filepath.Dir(p))), "apps") {
			return filepath.Dir(filepath.Dir(filepath.Dir(p)))
		}
		n := filepath.Dir(p)
		if n == p {
			break
		}
		p = n
	}
	h, _ := os.UserHomeDir()
	return filepath.Join(h, "scoop")
}
func sources(root string) ([]Source, error) {
	base := filepath.Join(root, "apps")
	xs, e := os.ReadDir(base)
	if e != nil {
		return nil, e
	}
	var a []Source
	for _, x := range xs {
		if !x.IsDir() || x.Name() == "dotnet-unified" {
			continue
		}
		r := filepath.Join(base, x.Name(), "current")
		if exists(filepath.Join(r, "sdk")) || exists(filepath.Join(r, "shared", "Microsoft.NETCore.App")) || exists(filepath.Join(r, "host", "fxr")) {
			a = append(a, Source{x.Name(), r})
		}
	}
	sort.Slice(a, func(i, j int) bool { return a[i].Name < a[j].Name })
	return a, nil
}
func plan(a []Source) map[string]string {
	m := map[string]string{}
	for _, s := range a {
		add := func(rel string) {
			if _, ok := m[rel]; !ok {
				m[rel] = filepath.Join(s.Root, rel)
			}
		}
		for _, section := range []string{"sdk", "host/fxr", "sdk-manifests", "templates", "template-packs", "library-packs", "metadata/workloads"} {
			for _, v := range dirs(filepath.Join(s.Root, filepath.FromSlash(section))) {
				add(filepath.Join(filepath.FromSlash(section), v))
			}
		}
		for _, section := range []string{"shared", "packs"} {
			for _, family := range dirs(filepath.Join(s.Root, section)) {
				for _, v := range dirs(filepath.Join(s.Root, section, family)) {
					add(filepath.Join(section, family, v))
				}
			}
		}
	}
	return m
}
func state(path string) (State, error) {
	s := State{Links: map[string]string{}}
	b, e := os.ReadFile(path)
	if errors.Is(e, os.ErrNotExist) {
		return s, nil
	}
	if e != nil {
		return s, e
	}
	var container struct {
		Links json.RawMessage `json:"links"`
	}
	if e = json.Unmarshal(b, &container); e != nil {
		return s, e
	}
	if len(container.Links) == 0 || string(container.Links) == "null" {
		return s, nil
	}
	if container.Links[0] == '{' {
		e = json.Unmarshal(container.Links, &s.Links)
		if s.Links == nil {
			s.Links = map[string]string{}
		}
		return s, e
	}
	if container.Links[0] == '[' {
		var legacy []string
		if e = json.Unmarshal(container.Links, &legacy); e != nil {
			return s, e
		}
		// Older PowerShell releases tracked only relative link names, not destinations.
		// The sentinel forces an update of tracked links without deleting untracked paths.
		for _, name := range legacy {
			s.Links[name] = "__legacy__"
		}
		return s, nil
	}
	return s, fmt.Errorf("unsupported links state format: %s", path)
}
func islink(p string) bool { i, e := os.Lstat(p); return e == nil && i.Mode()&os.ModeSymlink != 0 }
func link(p, t string) error {
	if e := os.Symlink(t, p); e == nil {
		return nil
	}
	c := exec.Command("cmd.exe", "/d", "/c", "mklink", "/J", p, t)
	out, e := c.CombinedOutput()
	if e != nil {
		return fmt.Errorf("junction %s: %w: %s", p, e, out)
	}
	return nil
}
func score(a string) int {
	var x, y, z int
	fmt.Sscanf(a, "%d.%d.%d", &x, &y, &z)
	return x*1000000 + y*1000 + z
}
func host(a []Source) string {
	best := ""
	rank := -1
	for _, s := range a {
		if !exists(filepath.Join(s.Root, "dotnet.exe")) {
			continue
		}
		for _, v := range dirs(filepath.Join(s.Root, "host", "fxr")) {
			if score(v) > rank {
				rank = score(v)
				best = s.Root
			}
		}
	}
	return best
}
func sync(root string, src []Source, dry bool) error {
	if len(src) == 0 {
		return errors.New("no Scoop SDK/runtime packages found")
	}
	p := plan(src)
	old, e := state(filepath.Join(root, ".links.json"))
	if e != nil {
		return e
	}
	newState := State{map[string]string{}}
	var names []string
	for n := range p {
		names = append(names, n)
	}
	sort.Strings(names)
	fmt.Printf("Discovered %d packages and %d versioned paths\n", len(src), len(names))
	if !dry {
		if e = os.MkdirAll(root, 0755); e != nil {
			return e
		}
	}
	for _, name := range names {
		to := p[name]
		dest := filepath.Join(root, name)
		i, e := os.Lstat(dest)
		if e == nil {
			if !islink(dest) {
				fmt.Println("SKIP unmanaged:", name)
				continue
			}
			prior, ok := old.Links[name]
			if !ok {
				fmt.Println("SKIP unknown link:", name)
				continue
			}
			if prior == to {
				newState.Links[name] = to
				continue
			}
			_ = i
			fmt.Println("RELINK", name)
			if !dry {
				if e = os.Remove(dest); e != nil {
					return e
				}
			}
		} else if !errors.Is(e, os.ErrNotExist) {
			return e
		} else {
			fmt.Println("LINK", name)
		}
		if !dry {
			if e = os.MkdirAll(filepath.Dir(dest), 0755); e != nil {
				return e
			}
			if e = link(dest, to); e != nil {
				return e
			}
		}
		newState.Links[name] = to
	}
	for n := range old.Links {
		if _, ok := newState.Links[n]; ok {
			continue
		}
		if _, ok := p[n]; ok {
			continue
		}
		path := filepath.Join(root, n)
		if islink(path) {
			fmt.Println("REMOVE stale", n)
			if !dry {
				if e = os.Remove(path); e != nil {
					return e
				}
			}
		}
	}
	h := host(src)
	if h == "" {
		return errors.New("no dotnet.exe host found")
	}
	fmt.Println("HOST", h)
	if !dry {
		from, e := os.Open(filepath.Join(h, "dotnet.exe"))
		if e != nil {
			return e
		}
		defer from.Close()
		tmp := filepath.Join(root, "dotnet.exe.tmp")
		out, e := os.Create(tmp)
		if e != nil {
			return e
		}
		_, e = io.Copy(out, from)
		ce := out.Close()
		if e != nil {
			return e
		}
		if ce != nil {
			return ce
		}
		_ = os.Remove(filepath.Join(root, "dotnet.exe"))
		if e = os.Rename(tmp, filepath.Join(root, "dotnet.exe")); e != nil {
			return e
		}
		b, _ := json.MarshalIndent(newState, "", "  ")
		if e = os.WriteFile(filepath.Join(root, ".links.json"), b, 0644); e != nil {
			return e
		}
	}
	if dry {
		fmt.Println("Preview only; no changes made")
	}
	return nil
}
func main() {
	exe, e := os.Executable()
	if e != nil {
		panic(e)
	}
	root := scoop(exe)
	dest := filepath.Join(filepath.Dir(exe), "data")
	args := os.Args[1:]
	cmd := "help"
	if len(args) > 0 {
		cmd = args[0]
		args = args[1:]
	}
	dry := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--dry-run", "-DryRun":
			dry = true
		case "--scoop-root":
			if i+1 < len(args) {
				i++
				root = args[i]
			}
		case "--root":
			if i+1 < len(args) {
				i++
				dest = args[i]
			}
		}
	}
	fail := func(e error) { fmt.Fprintln(os.Stderr, "ERROR:", e); os.Exit(1) }
	switch cmd {
	case "help", "--help":
		fmt.Println("dotnet-unified v0.3.1\nstatus | sync [--dry-run] | doctor | version")
	case "version":
		fmt.Println("0.3.1")
	case "status":
		fmt.Println("Scoop:", root, "\nUnified:", dest)
		src, e := sources(root)
		if e != nil {
			fail(e)
		}
		for _, s := range src {
			fmt.Println(s.Name, "->", s.Root)
		}
		st, e := state(filepath.Join(dest, ".links.json"))
		if e != nil {
			fail(e)
		}
		fmt.Println("Tracked:", len(st.Links))
	case "sync":
		src, e := sources(root)
		if e != nil {
			fail(e)
		}
		if e = sync(dest, src, dry); e != nil {
			fail(e)
		}
	case "doctor":
		host := filepath.Join(dest, "dotnet.exe")
		if !exists(host) {
			fail(errors.New("run sync first"))
		}
		for _, flag := range []string{"--list-sdks", "--list-runtimes"} {
			c := exec.Command(host, flag)
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			c.Env = os.Environ()
			if e = c.Run(); e != nil {
				fail(e)
			}
		}
	default:
		fail(fmt.Errorf("unknown command %s", cmd))
	}
}
