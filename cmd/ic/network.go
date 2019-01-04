package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"text/template"
	"time"

	"github.com/abates/cli"
	"github.com/abates/insteon"
)

func init() {
	cmd := Commands.Register("network", "", "dump map", nil)
	cmd.Register("dumpjson", "", "dump JSON network map to stdout", mapDumpCmd)
	// TODO(Robert): implement this, refactoring mapGraphvizCmd
	// cmd.Register("check", "<json map>", "read JSON map and check for issues", mapCheckCmd)
	cmd.Register("graphviz", "<json map> <graphviz nodes>", "read JSON map and graphviz nodes, dump graphviz file", mapGraphvizCmd)
}

// linkRecord is an alias for insteon.LinkRecord.  Because
// insteon.LinkRecord implements MarshalText which takes precedence
// over the default Marshaling, so we need to encode as a different
// type to round trip.
type linkRecord insteon.LinkRecord

// extendedLinkRecord is used internally to pass linkRecords with
// additional data attached.
type extendedLinkRecord struct {
	linkRecord
	Ok bool // both ends exist
}
type linksKey struct {
	Source, Dest string
	Group        insteon.Group
}
type linksMap map[linksKey]extendedLinkRecord

func mapDumpCmd(args []string, next cli.NextFunc) (err error) {
	devices := make(map[string][]linkRecord)

	plmLinks, err := modem.Links()
	if err != nil {
		return err
	}

	for _, l := range plmLinks {
		time.Sleep(2 * time.Second)
		if _, ok := devices[l.Address.String()]; ok {
			continue // already seen it
		}

		var links []linkRecord
		dev, err := devConnect(modem.Network, l.Address)
		if err != nil {
			return err
		}

		if linkable, ok := dev.(insteon.LinkableDevice); ok {
			insteon.Log.Debugf("getting links from %v", l.Address)
			dbLinks, err := linkable.Links()
			if err != nil {
				return fmt.Errorf("can't retrieve links from %v: %v", l.Address, err)
			}

			for _, l := range dbLinks {
				// Don't include modem links or 00.00.00 in the map.
				// (These sometimes show up as Group 0, which is not allowed.)
				if l.Address == modem.Address() || l.Address == insteon.Address([3]byte{}) {
					continue
				}
				links = append(links, linkRecord(*l))
			}
		}
		insteon.Log.Debugf("done with %v", l.Address)
		devices[l.Address.String()] = links
	}

	bytes, err := json.MarshalIndent(&devices, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling JSON: %v", err)
	}

	fmt.Printf("%s\n", bytes)
	return nil
}

func mapGraphvizCmd(args []string, next cli.NextFunc) (err error) {
	f, err := os.Open(args[0])
	if err != nil {
		return fmt.Errorf("can't open JSON input %v: %v", args[0], err)
	}
	d := json.NewDecoder(f)
	devices := make(map[string][]extendedLinkRecord)
	err = d.Decode(&devices)
	if err != nil {
		return fmt.Errorf("can't decode JSON: %v", err)
	}

	allLinks := make(linksMap)

	// Build a map of linksKey -> link.
	for d, dls := range devices {
		for _, l := range dls {
			if l.Flags.Available() {
				continue // Link deleted, ignore.
			}
			allLinks[linksKey{d, l.Address.String(), l.Group}] = l
		}
	}

	// Process links, marking bidirectional ones as Ok.
	for k, l := range allLinks {
		revKey := linksKey{k.Dest, k.Source, l.Group}
		_, ok := allLinks[revKey]
		if !ok {
			insteon.Log.Infof("can't find link: %v", revKey)
			// doesn't exist, or cleaned up
			continue
		}
		if l.Flags.Controller() {
			insteon.Log.Infof("%v is OK!", revKey)
			l.Ok = true // Mark link as bidirectional.
			allLinks[k] = l
			// Delete the responder reverse link from the table so we
			// don't draw it.
			delete(allLinks, linksKey{k.Dest, k.Source, l.Group})
		}
	}

	// GraphViz nodes are paramerized to give the user control over
	// the labels and look of the output.
	var nodes []byte
	if len(args) > 1 {
		if _, err = os.Stat(args[1]); !os.IsNotExist(err) {
			nodes, err = ioutil.ReadFile(args[1])
			if err != nil {
				return fmt.Errorf("can't read grapviz nodes input %v: %v", args[1], err)
			}
		}
	}

	data := struct {
		Links linksMap
		Nodes string
	}{
		Links: allLinks,
		Nodes: string(nodes),
	}
	err = dotTmpl.Execute(os.Stdout, data)
	if err != nil {
		return err
	}

	return nil
}

var dotTmpl = template.Must(template.New("dot").Parse(`
digraph g {
node [shape=record];

graph [overlap = false];
rankdir=LR;
splines=spline;

{{.Nodes}}

{{ range $k, $l :=   .Links }}
"{{$k.Source}}":{{$l.Group}} -> "{{$k.Dest}}":{{$l.Group}}:c [label="{{$l.Group}}{{if not $l.Ok}} ({{$l.Flags}}){{end}}",color="{{if $l.Ok }}black{{else}}red{{end}}"]
{{- end }}
}
`))
