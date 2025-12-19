package api

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"strings"
)

func StartServer(port string) {
	// API Routes
	http.HandleFunc("/api/channels", HandleChannels)
	http.HandleFunc("/api/channels/reorder", HandleChannelReorder)
	http.HandleFunc("/api/import", HandleImport)
	http.HandleFunc("/api/export", HandleExport)
	http.HandleFunc("/api/contacts", HandleContacts)
	http.HandleFunc("/api/zones", HandleZones)
	http.HandleFunc("/api/zones/assign", HandleZoneAssignment)
	http.HandleFunc("/api/scanlists", HandleScanLists)

	http.HandleFunc("/api/scanlists/assign", HandleScanListAssignment)
	http.HandleFunc("/api/filter_lists", HandleFilterLists)
	http.HandleFunc("/api/ws", HandleWebSocket)

	// Static Files
	// We need to access the embedded FS.
	// Since embed is in this package now, we can use it directly.
	// Note: The path inside embed depends on where the go command is run?
	// Actually embed paths are relative to the file containing the directive.
	// So if api/server.go is in codeplugs/api/, we need to adjust the path?
	// frontend/dist is at root: codeplugs/frontend/dist.
	// So relative to api/server.go, it is ../frontend/dist.
	// Embed patterns must match files relative to the package directory.
	// So I cannot embed ../frontend/dist directly in standard Go embed unless I use a workaround or move the file.
	//
	// Workaround: Move the embed to main.go (root) and pass the FS to StartServer.
	// Or put a dummy file in root to embed?
	// Best practice: Keep the embed in main.go or a root package, and pass fs.FS.

	// I will remove the embed from here and accept it as an argument.
}

func StartServerWithFS(port string, distFS fs.FS) {
	// API Routes
	http.HandleFunc("/api/channels", HandleChannels)
	http.HandleFunc("/api/channels/reorder", HandleChannelReorder)
	http.HandleFunc("/api/import", HandleImport)
	http.HandleFunc("/api/export", HandleExport)
	http.HandleFunc("/api/contacts", HandleContacts)
	http.HandleFunc("/api/zones", HandleZones)
	http.HandleFunc("/api/zones/assign", HandleZoneAssignment)
	http.HandleFunc("/api/scanlists", HandleScanLists)

	http.HandleFunc("/api/scanlists/assign", HandleScanListAssignment)
	http.HandleFunc("/api/filter_lists", HandleFilterLists)
	http.HandleFunc("/api/ws", HandleWebSocket)

	// SPA Handler
	fileServer := http.FileServer(http.FS(distFS))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		f, err := distFS.Open(path)
		if err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// Fallback to index.html for unknown routes (SPA)
		f, err = distFS.Open("index.html")
		if err != nil {
			http.Error(w, "index.html missing", http.StatusInternalServerError)
			return
		}
		defer f.Close()

		stat, _ := f.Stat()
		http.ServeContent(w, r, "index.html", stat.ModTime(), f.(io.ReadSeeker))
	})

	fmt.Printf("Starting server on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
