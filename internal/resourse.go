package clicrawler

type ResourseNode struct{
	Resourse string `json:"resource"`
	Title string  `json:"title"`
	Links[] ResourseNode `json:"links"`
}