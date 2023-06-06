package kubernetes

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/logging"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	elbv2api "sigs.k8s.io/aws-load-balancer-controller/apis/elbv2/v1beta1"
)

func readFile(f *core.SourceFile) (runtime.Object, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode(f.Program(), nil, nil)
	log := zap.L().Sugar().With(logging.FileField(f))

	if err != nil {
		log.Debugf("Error while decoding YAML object. Err was: %s", err)
		return nil, err
	}
	return obj, nil
}

func readElbv2ApiFiles(f *core.SourceFile) (runtime.Object, error) {
	sch := runtime.NewScheme()
	err := elbv2api.SchemeBuilder.AddToScheme(sch)
	if err != nil {
		return nil, err
	}
	decode := serializer.NewCodecFactory(sch).UniversalDeserializer().Decode
	obj, _, err := decode(f.Program(), nil, nil)
	if err != nil {
		return nil, err
	}
	return obj, nil
}
