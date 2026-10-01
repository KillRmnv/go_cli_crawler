package clicrawler

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"log"
	"os"
)

type Storage interface {
	Save(data []ResourseNode) ([]byte,error)
}

type JSONStorage struct {
	filepath string
	logger log.Logger
}

func (s *JSONStorage) Save(result []ResourseNode) ([]byte,error) {
	data,err:=json.Marshal(result,jsontext.AllowInvalidUTF8(true))
	if err!=nil{
		return nil,err
	}
	s.logger.Println("Saving to file "+s.filepath)

	if err:=os.WriteFile(s.filepath,data,0644); err!=nil{
		s.logger.Println("Can not write result to "+s.filepath+":"+err.Error())
		return nil,err
	}
	return data,nil
}
