/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"encoding/json"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appv1 "github.com/ybooks240/api/v1"
)

var (
	oldSpecAnnotations = "old/spec"
)

// MyappReconciler reconciles a Myapp object
type MyappReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=app.github.com,resources=myapps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=app.github.com,resources=myapps/status,verbs=get;update;patch

func (r *MyappReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("myapp", req.NamespacedName)

	log.Info("Welcome to  Myapp-demo ")
	// your logic here
	// 首先获取Myapp的实例
	var myapp appv1.Myapp
	err := r.Client.Get(ctx, req.NamespacedName, &myapp)
	if err != nil {
		if client.IgnoreNotFound(err) != err {
			return ctrl.Result{}, err
		}
		// 在删除一个不存在的对象的时候，可能会报not-found的错误
		// 这种情况不需要重新入队列修复
		return ctrl.Result{}, nil
	}
	// 当前的对象标记为了删除
	if myapp.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	// 新建
	// 如果不存在关联的资源，是不是应该去创建
	// 如果存在关联的资源，是不是要去判断是否需要更新
	deploy := &appsv1.Deployment{}

	//fmt.Println(r.Client.Get(ctx, req.NamespacedName, deploy))
	if err := r.Client.Get(ctx, req.NamespacedName, deploy); err != nil && errors.IsNotFound(err) {

		// 关联Annotations
		data, err := json.Marshal(myapp.Spec)
		if err != nil {
			return ctrl.Result{}, err
		}
		log.Info("Annotations插入数据：", oldSpecAnnotations, string(data))
		if myapp.Annotations != nil {
			myapp.Annotations[oldSpecAnnotations] = string(data)
		} else {
			myapp.Annotations = map[string]string{
				oldSpecAnnotations: string(data),
			}
		}

		// 更新Annotations

		if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			return r.Client.Update(ctx, &myapp)
		}); err != nil {
			return ctrl.Result{}, err
		}

		// Deployment 不存在，创建关联的资源
		newDeploy := NewDeploy(&myapp)
		if err := r.Client.Create(ctx, newDeploy); err != nil {
			return ctrl.Result{}, err
		}
		newService := NewService(&myapp)
		if err := r.Client.Create(ctx, newService); err != nil {
			return ctrl.Result{}, err
		}
		// 创建成功
		return ctrl.Result{}, nil
	}

	// 更新
	// 判断是否需要更新
	//    	判断yaml文件是否发生变化
	oldSpec := appv1.MyappSpec{}
	//log.Info("json:", oldSpecAnnotations)
	if err := json.Unmarshal([]byte(myapp.Annotations[oldSpecAnnotations]), &oldSpec); err != nil {
		log.Error(err, "认真看错误")
		return ctrl.Result{}, err
	}
	// 获取成功old-spec，和新spec进行对比
	if !reflect.DeepEqual(myapp.Spec, oldSpec) {
		// spec 发现更新了，去更新关联资源
		newDeploy := NewDeploy(&myapp)
		oldDeply := &appsv1.Deployment{}
		if err := r.Client.Get(ctx, req.NamespacedName, oldDeply); err != nil {
			return ctrl.Result{}, err
		}
		oldDeply.Spec = newDeploy.Spec
		// 正常就应该直接去更新oldDeploy
		// 注意： 一般情况下不会直接调用update进行更新
		//r.Client.Update(ctx,oldDeply)
		if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			return r.Client.Update(ctx, oldDeply)
		}); err != nil {
			return ctrl.Result{}, err
		}

		// 更新service

		NewService := NewService(&myapp)

		oldService := corev1.Service{}
		if err := r.Client.Get(ctx, req.NamespacedName, &oldService); err != nil {
			return ctrl.Result{}, err
		}

		NewService.Spec.ClusterIP = oldService.Spec.ClusterIP
		oldService.Spec = NewService.Spec
		if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			return r.Client.Update(ctx, &oldService)
		}); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *MyappReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1.Myapp{}).
		Complete(r)
}
