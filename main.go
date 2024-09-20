package main

import (
    "fmt"
    "net/http"
    "html/template"
)

type Event struct {
    Name string
    Date string
}

var events []Event

func main() {
    http.HandleFunc("/", serveIndex)
    http.HandleFunc("/add-event", addEvent)

    fmt.Println("Server started at http://localhost:8080")
    http.ListenAndServe(":8080", nil)
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
    tmpl := `<html>
<head><title>Maturitní Slavnost Management</title></head>
<body>
    <h1>Maturitní Slavnost Management</h1>
    <form id="event-form" hx-post="/add-event" hx-target="#events-list" hx-swap="beforeend">
        <label for="event-name">Event Name:</label>
        <input type="text" id="event-name" name="name" required>
        
        <label for="event-date">Date:</label>
        <input type="date" id="event-date" name="date" required>
        
        <button type="submit">Add Event</button>
    </form>
    <ul id="events-list">
        {{range .}}
        <li>{{.Name}} - {{.Date}}</li>
        {{end}}
    </ul>
</body>
</html>`

    t, err := template.New("index").Parse(tmpl)
    if err != nil {
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }

    t.Execute(w, events)
}

func addEvent(w http.ResponseWriter, r *http.Request) {
    if r.Method == http.MethodPost {
        name := r.FormValue("name")
        date := r.FormValue("date")

        event := Event{Name: name, Date: date}
        events = append(events, event)

        // Return new event as HTML
        fmt.Fprintf(w, "<li>%s - %s</li>", name, date)
    } else {
        http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
    }
}
