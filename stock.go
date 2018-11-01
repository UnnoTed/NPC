package main

import (
	"bytes"
	"errors"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	av "github.com/cmckee-dev/go-alpha-vantage"
	"github.com/olekukonko/tablewriter"
	"github.com/rs/zerolog/log"
)

var brt *time.Location

func init() {
	var err error
	brt, err = time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		panic(err)
	}
}

type stockName string

const (
	stockNameIbovespa stockName = "Ibovespa"
	stockNameDolar    stockName = "Dolar"
	stockNameEuro     stockName = "Euro"
)

type stock struct {
	ID    stockName
	Name  string
	Code  string
	Data  []*av.TimeSeriesValue
	Daily []*av.TimeSeriesValue
}

type infoConfig struct {
	Detailed bool
	Table    bool // ascii
	Colored  bool // as css
	Daily    bool
	Max      int
}

type byTime []*av.TimeSeriesValue

func (a byTime) Len() int           { return len(a) }
func (a byTime) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byTime) Less(i, j int) bool { return a[i].Time.After(a[j].Time) }

func (s *stock) get(interval ...av.TimeInterval) error {
	if s.Code == "" {
		return errors.New("Code is missing")
	}

	if len(interval) == 0 {
		interval = append(interval, av.TimeIntervalSixtyMinute)
	}

	stsi, err := avc.StockTimeSeriesIntraday(interval[0], s.Code)
	if err != nil {
		return err
	}

	s.Data = stsi
	sort.Sort(byTime(s.Data))

	go func(s *stock) {
		time.Sleep(time.Minute)

		sts, err := avc.StockTimeSeries(av.TimeSeriesDaily, s.Code)
		if err != nil {
			log.Error().Err(err).Msg("Error getting diary stock")
		}

		s.Daily = sts
		sort.Sort(byTime(s.Daily))
		db.Save(s)
	}(s)

	return db.Save(s)
}

func (s *stock) status() string {
	var status string
	if len(s.Data) > 0 {
		status += s.Name[:1] + "="

		v := strconv.FormatFloat(s.Data[0].Close, 'f', -1, 64)
		if len(v) >= 4 && s.ID != stockNameIbovespa {
			status += v[:4]

		} else {
			ibvsp := strings.Split(v, ".")
			if len(ibvsp) > 0 {
				status += ibvsp[0]
			}
		}
	}

	return status
}

func (s *stock) info(c *infoConfig) string {
	if len(s.Data) == 0 {
		return "Erro: Dados " + s.Name + " indisponiveis."
	}

	var data [][]string
	for i := 0; i < c.Max; i++ {
		var (
			actual *av.TimeSeriesValue
			old    *av.TimeSeriesValue
			info   []string
		)

		if c.Daily {
			actual = s.Daily[i]
			old = s.Daily[i+1]
		} else {
			actual = s.Data[i]
			old = s.Data[i+1]
		}

		fechou := strconv.FormatFloat(actual.Close, 'f', -1, 64)
		abriu := strconv.FormatFloat(actual.Open, 'f', -1, 64)
		baixa := strconv.FormatFloat(actual.Low, 'f', -1, 64)
		alta := strconv.FormatFloat(actual.High, 'f', -1, 64)

		// info = append(info, s.Name)
		info = append(info, abriu)
		info = append(info, fechou)
		info = append(info, baixa)
		info = append(info, alta)
		info = append(info, actual.Time.In(brt).Format("02/01/2006 15:04"))
		info = append(info, Time(actual.Time.In(brt)))

		val := 0.0
		diff := ""
		subiu := actual.Close > old.Close

		if subiu {
			inc := actual.Close - old.Close
			val = inc / old.Close * 100
			diff = "+"

		} else {
			dec := old.Close - actual.Close
			val = dec / old.Close * 100
			diff = "-"
		}

		diff += strconv.FormatFloat(val, 'f', 2, 64)
		if len(diff) > 2 && diff[2] == '-' {
			diff = diff[:2] + diff[3:]
		}

		info = append(info, diff+"%")
		data = append(data, info)
	}

	buf := bytes.NewBuffer(nil)
	if c.Colored {
		buf.WriteString("```css\n")
	}

	if c.Table {
		if !c.Colored {
			buf.WriteString("```\n")
		}

		table := tablewriter.NewWriter(buf)
		table.SetHeader([]string{"Abriu", "Fechou", "Baixa", "Alta", "Data", "Tempo", "Diff"})
		table.SetAutoMergeCells(true)
		table.SetRowLine(true)

		intervalo := "1h"
		if c.Daily {
			intervalo = "24h"
		}

		table.SetFooter([]string{s.Name, "", "", "", "Intervalo: " + intervalo, "", ""})
		table.SetCenterSeparator("┼")
		table.SetRowSeparator("─")
		table.SetAlignment(tablewriter.ALIGN_LEFT)
		table.AppendBulk(data)
		table.Render()
	} else {
		for _, line := range data {
			buf.WriteString(strings.Join(line, " ") + "\n")
		}
	}

	if c.Table || c.Colored {
		buf.WriteString("```")
	}

	return buf.String()
}

func round(x, unit float64) float64 {
	return math.Round(x/unit) * unit
}

func parseInfoConfig(cmds []string) *infoConfig {
	cfg := &infoConfig{
		Colored: true,
		Table:   true,
		Daily:   false,
		Max:     5,
	}

	changedColored := false
	changedTable := false
	changedDaily := false
	changedMax := false

	for _, cmd := range cmds {
		log.Print(cmd)

		if !changedColored {
			if i := parseInfoCmd(cmd, "colorido"); i >= 1 {
				cfg.Colored = i == 1
				changedColored = true
				continue
			}
		}

		if !changedTable {
			if i := parseInfoCmd(cmd, "tabela"); i >= 1 {
				cfg.Table = i == 1
				changedTable = true
				continue
			}
		}

		if !changedDaily {
			i := parseInfoCmd(cmd, "diario")
			if i >= 1 || cmd == "diario" {
				cfg.Daily = (i == 1 || cmd == "diario")
				changedDaily = true
				continue
			}
		}

		if !changedMax {
			if strings.HasPrefix(cmd, "max=") {
				mstr := strings.TrimPrefix(cmd, "max=")
				max, err := strconv.Atoi(mstr)
				if err != nil {
					continue
				}

				cfg.Max = max
				changedMax = true
				continue
			}
		}
	}

	return cfg
}

func parseInfoCmd(cmd string, expected string) int {
	if strings.HasPrefix(cmd, expected+"=") {
		modo := strings.TrimPrefix(cmd, expected+"=")

		if modo == "sim" {
			return 1
		}

		return 2
	}

	return 0
}
