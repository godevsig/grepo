module github.com/godevsig/grepo

go 1.18

require (
	github.com/go-echarts/go-echarts/v2 v2.2.7
	github.com/godevsig/adaptiveservice v0.9.26-0.20230830014831-c95dfbc18882
	github.com/godevsig/glib v0.1.2-0.20230830021401-ee447d68739c
	github.com/gorilla/mux v1.8.0
	github.com/niubaoshu/gotiny v0.0.3
)

require github.com/barkimedes/go-deepcopy v0.0.0-20220514131651-17c30cfc62df // indirect

replace (
	github.com/go-echarts/go-echarts/v2 => github.com/godevsig/go-echarts/v2 v2.0.0-20211101104447-e8e4a51bc4fd
	github.com/niubaoshu/gotiny => github.com/godevsig/gotiny v0.0.4-0.20210913173728-083dd4b72177
)
