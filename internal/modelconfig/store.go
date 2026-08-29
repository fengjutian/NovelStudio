package modelconfig

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type Provider struct { BaseURL string `json:"baseUrl"`; APIKey string `json:"apiKey"`; Model string `json:"model"`; Enabled bool `json:"enabled"` }
type Config struct { ActiveProvider string `json:"activeProvider"`; DeepSeek Provider `json:"deepseek"`; MiniMax Provider `json:"minimax"` }
type PublicProvider struct { BaseURL string `json:"baseUrl"`; Model string `json:"model"`; Enabled bool `json:"enabled"`; HasAPIKey bool `json:"hasApiKey"` }
type PublicConfig struct { ActiveProvider string `json:"activeProvider"`; DeepSeek PublicProvider `json:"deepseek"`; MiniMax PublicProvider `json:"minimax"`; Path string `json:"path"` }
type UpdateProvider struct { BaseURL string `json:"baseUrl"`; APIKey string `json:"apiKey"`; Model string `json:"model"`; Enabled bool `json:"enabled"`; ClearAPIKey bool `json:"clearApiKey"` }
type UpdateInput struct { ActiveProvider string `json:"activeProvider"`; DeepSeek UpdateProvider `json:"deepseek"`; MiniMax UpdateProvider `json:"minimax"` }
type Store struct{ Path string }

func Default() Config { return Config{ActiveProvider:"deepseek",DeepSeek:Provider{BaseURL:"https://api.deepseek.com",Model:"deepseek-v4-flash"},MiniMax:Provider{BaseURL:"https://api.minimaxi.com/v1",Model:"MiniMax-M2.7"}} }
func (s Store) Load()(Config,error){config:=Default();raw,err:=os.ReadFile(s.Path);if errors.Is(err,os.ErrNotExist){return config,nil};if err!=nil{return Config{},err};if err=json.Unmarshal(raw,&config);err!=nil{return Config{},err};return config,nil}
func(s Store)Public()(PublicConfig,error){config,err:=s.Load();if err!=nil{return PublicConfig{},err};return PublicConfig{ActiveProvider:config.ActiveProvider,DeepSeek:public(config.DeepSeek),MiniMax:public(config.MiniMax),Path:s.Path},nil}
func(s Store)Save(input UpdateInput)(PublicConfig,error){current,err:=s.Load();if err!=nil{return PublicConfig{},err};active:=strings.ToLower(strings.TrimSpace(input.ActiveProvider));if active!="deepseek"&&active!="minimax"{return PublicConfig{},errors.New("activeProvider must be deepseek or minimax")};current.ActiveProvider=active;current.DeepSeek=merge(current.DeepSeek,input.DeepSeek);current.MiniMax=merge(current.MiniMax,input.MiniMax);selected:=current.DeepSeek;if active=="minimax"{selected=current.MiniMax};if selected.Enabled&&(strings.TrimSpace(selected.BaseURL)==""||strings.TrimSpace(selected.Model)==""||strings.TrimSpace(selected.APIKey)==""){return PublicConfig{},errors.New("enabled active provider requires baseUrl, model and apiKey")};if err:=os.MkdirAll(filepath.Dir(s.Path),0700);err!=nil{return PublicConfig{},err};raw,_:=json.MarshalIndent(current,"","  ");temporary:=s.Path+".tmp";if err:=os.WriteFile(temporary,raw,0600);err!=nil{return PublicConfig{},err};if err:=os.Rename(temporary,s.Path);err!=nil{return PublicConfig{},err};return s.Public()}
func(s Store)Active()(string,Provider,error){config,err:=s.Load();if err!=nil{return "",Provider{},err};if config.ActiveProvider=="minimax"{return "minimax",config.MiniMax,nil};return "deepseek",config.DeepSeek,nil}
func merge(current Provider,input UpdateProvider)Provider{current.BaseURL=strings.TrimRight(strings.TrimSpace(input.BaseURL),"/");current.Model=strings.TrimSpace(input.Model);current.Enabled=input.Enabled;if input.ClearAPIKey{current.APIKey=""}else if strings.TrimSpace(input.APIKey)!=""{current.APIKey=strings.TrimSpace(input.APIKey)};return current}
func public(value Provider)PublicProvider{return PublicProvider{BaseURL:value.BaseURL,Model:value.Model,Enabled:value.Enabled,HasAPIKey:value.APIKey!=""}}
