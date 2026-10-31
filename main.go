package main

import (
	"archive/zip"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"os"
	"path/filepath"
)

func GenCode(length int) string {
	charset := "QWERTYUIOPASDFGHJKLZXCVBNM"

	code := make([]byte, length)
	for i := range length {
		code[i] = charset[rand.IntN(len(charset))]
	}

	return string(code)
}

func main() {
	paths := os.Args[1:]

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	if len(paths) <= 0 {
		logger.Info("Nothing to do")
		return
	}

	type Path = string
	filepaths := make([]Path, 0, 10)

	for _, p := range paths {
		stat, err := os.Stat(p)

		if err != nil {
			logger.Warn("failed to retrieve stat file, ignoring.",
				"path", p,
				"reason", err,
			)

			continue
		}

		if stat.IsDir() {
			logger.Warn("path is a directory, ignoring",
				"path", p,
			)
			continue
		}

		filepaths = append(filepaths, p)
	}

	code := GenCode(4)

	logger.Info("Code acquired",
		"code", code,
	)

	http.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		file, err := os.Open("index.html")

		if err != nil {
			logger.Error("Failed to read index.html",
				"reason", err,
			)

			http.Error(w, "Failed to read index.html", http.StatusInternalServerError)
			return
		}

		stat, err := file.Stat()

		if err != nil {
			logger.Error("Failed to get stat file",
				"reason", err,
			)
		}

		defer file.Close()

		r.Header.Set("content-length", string(stat.Size()))

		if _, err := io.Copy(w, file); err != nil {
			logger.Error("Failed to send html file",
				"reason", err,
			)
		}
	})

	http.HandleFunc("GET /{code}", func(w http.ResponseWriter, r *http.Request) {
		reqcode := r.PathValue("code")

		if reqcode == "" {
			http.Error(w, "code shouldn't be empty", http.StatusBadRequest)
			return
		}

		if reqcode != code {
			http.Error(w, "code isn't the same", http.StatusUnauthorized)
			return

		}

		zipwriter := zip.NewWriter(w)
		defer zipwriter.Close()

		r.Header.Set("content-disposition", "attachment; filename="+code+".zip")

		for _, fp := range filepaths {
			writer, err := zipwriter.Create(filepath.Base(fp))

			if err != nil {
				http.Error(w, "Failed to create a file writer for zip", http.StatusInternalServerError)
				return
			}

			f, err := os.Open(fp)

			if err != nil {
				logger.Error("Failed to open file",
					"reason", err,
				)
				http.Error(w, "Failed to open file", http.StatusInternalServerError)
				return
			}

			if _, err := io.Copy(writer, f); err != nil {
				logger.Error("Failed to stream file",
					"reason", err,
				)

				http.Error(w, "Failed to stream file", http.StatusInternalServerError)

				f.Close()
				return
			}

			f.Close()
		}
	})

	addrress := ":8080"

	logger.Info("Starting HTTP server",
		"address", addrress,
	)

	if err := http.ListenAndServe(addrress, nil); err != nil {
		logger.Error("Failed to start HTTP server",
			"reason", err,
		)
	}
}
