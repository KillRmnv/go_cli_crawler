package clicrawler

type ResourseNode struct{
	Resourse string
	Title string
	Links[] ResourseNode
}