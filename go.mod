module github.com/godevsig/grepo

go 1.16

require (
	github.com/go-echarts/go-echarts/v2 v2.0.0-20210921152819-048776e902c7
	github.com/godevsig/adaptiveservice v0.9.24-0.20230329141516-d104c403854d
	github.com/godevsig/glib v0.0.0-20230815095222-ea0790ec1ec1
	github.com/gorilla/mux v1.8.0
	github.com/niubaoshu/gotiny v0.0.3
)

replace (
	github.com/go-echarts/go-echarts/v2 => github.com/godevsig/go-echarts/v2 v2.0.0-20211101104447-e8e4a51bc4fd
	github.com/niubaoshu/gotiny => github.com/godevsig/gotiny v0.0.4-0.20210913173728-083dd4b72177
)
