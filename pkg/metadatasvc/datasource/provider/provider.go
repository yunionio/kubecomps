package provider

import "fmt"

// FormatURL is the meta-data service url as a format string
// to be replaced by version.
// Example:
// var u FormatURL = "http://169.254.169.254/metadata/%s/%s"
// fmt.Printf(u.Fill("latest", "hostname")) -> http://169.254.169.254/metadata/latest/hostname
type FormatURL string

// Fill method is a wrapper around fmt.Sprintf to fill format URL
func (u FormatURL) Fill(vals ...interface{}) string {
	return fmt.Sprintf(string(u), vals...)
}
