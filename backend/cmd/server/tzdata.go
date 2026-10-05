package main

// Embed the IANA timezone database: the distroless image has none, and auth validates timezones.
import _ "time/tzdata"
