package main

import (
	"math"
	"time"

	humanize "github.com/dustin/go-humanize"
)

var magnitudes = []humanize.RelTimeMagnitude{
	{
		D:      time.Second,
		Format: "agora",
		DivBy:  time.Second,
	},
	{
		D:      2 * time.Second,
		Format: "1 segundo %s",
		DivBy:  1,
	},
	{
		D:      time.Minute,
		Format: "%d segundos %s",
		DivBy:  time.Second,
	},
	{
		D:      2 * time.Minute,
		Format: "1 minuto %s",
		DivBy:  1,
	},
	{
		D:      time.Hour,
		Format: "%d minutos %s",
		DivBy:  time.Minute,
	},
	{
		D:      2 * time.Hour,
		Format: "1 hora %s",
		DivBy:  1,
	},
	{
		D:      humanize.Day,
		Format: "%d horas %s",
		DivBy:  time.Hour,
	},
	{
		D:      2 * humanize.Day,
		Format: "1 dia %s",
		DivBy:  1,
	},
	{
		D:      humanize.Week,
		Format: "%d dias %s",
		DivBy:  humanize.Day,
	},
	{
		D:      2 * humanize.Week,
		Format: "1 semana %s",
		DivBy:  1,
	},
	{
		D:      humanize.Month,
		Format: "%d semanas %s",
		DivBy:  humanize.Week,
	},
	{
		D:      2 * humanize.Month,
		Format: "1 mês %s",
		DivBy:  1,
	},
	{
		D:      humanize.Year,
		Format: "%d meses %s",
		DivBy:  humanize.Month,
	},
	{
		D:      18 * humanize.Month,
		Format: "1 ano %s",
		DivBy:  1,
	},
	{
		D:      2 * humanize.Year,
		Format: "2 anos %s",
		DivBy:  1,
	},
	{
		D:      humanize.LongTime,
		Format: "%d anos %s",
		DivBy:  humanize.Year,
	},
	{
		D:      math.MaxInt64,
		Format: "muito tempo %s",
		DivBy:  1,
	},
}

func Time(then time.Time) string {
	return humanize.CustomRelTime(then, time.Now(), "atrás", "em", magnitudes)
}
