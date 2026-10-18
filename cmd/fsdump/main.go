package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/firestore"

	"booking/config"
	pkgfirestore "booking/pkg/firestore"
)

// это короче вспомогательный тулл
func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: fsdump <collection or document path>")
		os.Exit(2)
	}

	err := run(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(path string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config.Load: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := pkgfirestore.New(ctx, cfg.FirestoreProjectID)
	if err != nil {
		return fmt.Errorf("firestore.New: %w", err)
	}
	defer client.Close()

	path = strings.Trim(path, "/")

	var snaps []*firestore.DocumentSnapshot
	if strings.Count(path, "/")%2 == 1 {
		snap, err := client.Doc(path).Get(ctx)
		if err != nil {
			return err
		}
		snaps = append(snaps, snap)
	} else {
		snaps, err = client.Collection(path).Documents(ctx).GetAll()
		if err != nil {
			return err
		}
	}

	for _, snap := range snaps {
		data, err := json.MarshalIndent(snap.Data(), "", "  ")
		if err != nil {
			return err
		}
		_, name, _ := strings.Cut(snap.Ref.Path, "/documents/")
		fmt.Printf("%s\n%s\n\n", name, data)
	}

	return nil
}
