module github.com/matt-tyler/elasticsearch-operator

require (
	cloud.google.com/go v0.26.0
	github.com/NYTimes/gziphandler v1.0.1 // indirect
	github.com/PuerkitoBio/purell v1.1.0
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578
	github.com/asaskevich/govalidator v0.0.0-20180720115003-f9ffefc3facf // indirect
	github.com/beorn7/perks v0.0.0-20180321164747-3a771d992973 // indirect
	github.com/coreos/etcd v3.3.9+incompatible // indirect
	github.com/coreos/go-semver v0.2.0 // indirect
	github.com/coreos/go-systemd v0.0.0-20180828140353-eee3db372b31 // indirect
	github.com/davecgh/go-spew v1.1.0
	github.com/docker/docker v1.13.1 // indirect
	github.com/elazarl/go-bindata-assetfs v1.0.0 // indirect
	github.com/emicklei/go-restful v2.4.0+incompatible
	github.com/emicklei/go-restful-swagger12 v0.0.0-20170926063155-7524189396c6 // indirect
	github.com/evanphx/json-patch v4.1.0+incompatible // indirect
	github.com/fsnotify/fsnotify v1.4.7
	github.com/ghodss/yaml v1.0.0
	github.com/globalsign/mgo v0.0.0-20180905125535-1ca0a4f7cbcb // indirect
	github.com/go-openapi/analysis v0.0.0-20180825180245-b006789cd277 // indirect
	github.com/go-openapi/errors v0.0.0-20180926063021-1801eed82180 // indirect
	github.com/go-openapi/jsonpointer v0.0.0-20170102174223-779f45308c19
	github.com/go-openapi/jsonreference v0.0.0-20161105162150-36d33bfe519e
	github.com/go-openapi/loads v0.0.0-20180825180312-fd899182a268 // indirect
	github.com/go-openapi/runtime v0.0.0-20180920151709-4f900dc2ade9 // indirect
	github.com/go-openapi/spec v0.0.0-20171219195406-fa03337d7da5
	github.com/go-openapi/strfmt v0.0.0-20180910212104-776114108ccc // indirect
	github.com/go-openapi/swag v0.0.0-20180102232305-84f4bee7c0a6
	github.com/go-openapi/validate v0.0.0-20180825180342-e0648ff40507 // indirect
	github.com/gogo/protobuf v0.0.0-20171007142547-342cbe0a0415
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/groupcache v0.0.0-20171101203131-84a468cf14b4
	github.com/golang/protobuf v1.2.0
	github.com/google/btree v0.0.0-20161217183710-316fb6d3f031
	github.com/google/gofuzz v0.0.0-20170612174753-24818f796faf
	github.com/googleapis/gnostic v0.1.0
	github.com/gregjones/httpcache v0.0.0-20171119193500-2bcd89a1743f
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0 // indirect
	github.com/hashicorp/golang-lru v0.0.0-20160813221303-0a025b7e63ad
	github.com/hashicorp/hcl v0.0.0-20171017181929-23c074d0eceb
	github.com/howeyc/gopass v0.0.0-20170109162249-bf9dde6d0d2c
	github.com/imdario/mergo v0.0.0-20171009183408-7fe0c75c13ab
	github.com/inconshreveable/mousetrap v1.0.0
	github.com/json-iterator/go v1.1.5
	github.com/juju/ratelimit v1.0.1
	github.com/magiconair/properties v1.7.4
	github.com/mailru/easyjson v0.0.0-20171120080333-32fa128f234d
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/mitchellh/mapstructure v0.0.0-20171017171808-06020f85339e
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/onsi/ginkgo v1.4.0
	github.com/onsi/gomega v1.3.0
	github.com/pborman/uuid v1.2.0 // indirect
	github.com/pelletier/go-toml v1.0.1
	github.com/petar/GoLLRB v0.0.0-20130427215148-53be0d36a84c
	github.com/peterbourgon/diskv v2.0.1+incompatible
	github.com/prometheus/client_golang v0.8.0 // indirect
	github.com/prometheus/client_model v0.0.0-20180712105110-5c3871d89910 // indirect
	github.com/prometheus/common v0.0.0-20180801064454-c7de2306084e // indirect
	github.com/prometheus/procfs v0.0.0-20180920065004-418d78d0b9a7 // indirect
	github.com/spf13/afero v1.0.1
	github.com/spf13/cast v1.1.0
	github.com/spf13/cobra v0.0.0-20171231101556-b95ab734e27d
	github.com/spf13/jwalterweatherman v0.0.0-20180109140146-7c0cea34c8ec
	github.com/spf13/pflag v1.0.0
	github.com/spf13/viper v0.0.0-20171227194143-aafc9e6bc7b7
	github.com/ugorji/go/codec v0.0.0-20180920200043-2c86ab75771c // indirect
	go.uber.org/atomic v1.3.1
	go.uber.org/multierr v1.1.0
	go.uber.org/zap v1.7.1
	golang.org/x/crypto v0.0.0-20180110100327-6bd909f163c8
	golang.org/x/net v0.0.0-20180826012351-8a410e7b638d
	golang.org/x/oauth2 v0.0.0-20180821212333-d2e6202438be
	golang.org/x/sys v0.0.0-20180830151530-49385e6e1522
	golang.org/x/text v0.3.0
	golang.org/x/time v0.0.0-20180412165947-fbb02b2291d2 // indirect
	google.golang.org/api v0.0.0-20180110000348-aa8d5015340e
	google.golang.org/appengine v1.1.0
	google.golang.org/grpc v1.15.0 // indirect
	gopkg.in/inf.v0 v0.9.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
	gopkg.in/yaml.v2 v2.0.0
	k8s.io/api v0.0.0-20180925152912-a191abe0b71e
	k8s.io/apiextensions-apiserver v0.0.0-20180905004947-16750353bf97
	k8s.io/apimachinery v0.0.0-20180904193909-def12e63c512
	k8s.io/apiserver v0.0.0-20180926233834-7010da59f6e6 // indirect
	k8s.io/client-go v8.0.0+incompatible
	k8s.io/code-generator v0.0.0-20180904193909-8c97d6ab64da // indirect
	k8s.io/kube-openapi v0.0.0-20180108222231-a07b7bbb58e7
)
