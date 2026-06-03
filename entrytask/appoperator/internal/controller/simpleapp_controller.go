package controller

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	appv1 "github.com/keyao/appoperator/api/v1"
)

// SimpleAppReconciler reconciles a SimpleApp object
type SimpleAppReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=app.example.com,resources=simpleapps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=app.example.com,resources=simpleapps/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete

func (r *SimpleAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// 1. 获取 SimpleApp 实例
	app := &appv1.SimpleApp{}
	if err := r.Get(ctx, req.NamespacedName, app); err != nil {
		log.Error(err, "无法获取 SimpleApp 资源")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 设置默认副本数为 1
	if app.Spec.Replicas == nil {
		defaultReplicas := int32(1)
		app.Spec.Replicas = &defaultReplicas
	}

	// 设置默认服务类型为 ClusterIP
	if app.Spec.ServiceType == "" {
		app.Spec.ServiceType = corev1.ServiceTypeClusterIP
	}

	// 2. 处理 Deployment 资源
	deploy := r.createDeployment(app)
	if err := controllerutil.SetControllerReference(app, deploy, r.Scheme); err != nil {
		log.Error(err, "设置 Deployment 所有者引用失败")
		return ctrl.Result{}, err
	}

	foundDeploy := &appsv1.Deployment{}
	//去集群查询Deployment
	err := r.Get(ctx, types.NamespacedName{Name: deploy.Name, Namespace: deploy.Namespace}, foundDeploy)
	if err != nil { //只代表出错了，不能确定是没找到资源
		if errors.IsNotFound(err) { //没找到资源
			log.Info("创建 Deployment", "name", deploy.Name)
			if err := r.Create(ctx, deploy); err != nil {
				log.Error(err, "创建 Deployment 失败")
				return ctrl.Result{}, err
			}
		} else {
			log.Error(err, "获取 Deployment 失败")
			return ctrl.Result{}, err
		}
	} else {
		// 更新 Deployment
		needsUpdate := false
		if *foundDeploy.Spec.Replicas != *app.Spec.Replicas {
			foundDeploy.Spec.Replicas = app.Spec.Replicas
			needsUpdate = true
		}
		if foundDeploy.Spec.Template.Spec.Containers[0].Image != app.Spec.Image {
			foundDeploy.Spec.Template.Spec.Containers[0].Image = app.Spec.Image
			needsUpdate = true
		}
		if foundDeploy.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort != app.Spec.Port {
			foundDeploy.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort = app.Spec.Port
			needsUpdate = true
		}
		if needsUpdate {
			log.Info("更新 Deployment", "name", foundDeploy.Name)
			//将本地更新之后的founddeploy同步到Kubernetes 集群里面去
			if err := r.Update(ctx, foundDeploy); err != nil {
				log.Error(err, "更新 Deployment 失败")
				return ctrl.Result{}, err
			}
		}
	}

	// 3. 处理 Service 资源
	svc := r.createService(app)
	if err := controllerutil.SetControllerReference(app, svc, r.Scheme); err != nil {
		log.Error(err, "设置 Service 所有者引用失败")
		return ctrl.Result{}, err
	}

	foundSvc := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: svc.Name, Namespace: svc.Namespace}, foundSvc)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("创建 Service", "name", svc.Name)
			if err := r.Create(ctx, svc); err != nil {
				log.Error(err, "创建 Service 失败")
				return ctrl.Result{}, err
			}
		} else {
			log.Error(err, "获取 Service 失败")
			return ctrl.Result{}, err
		}
	} else {
		// 更新 Service
		needsUpdate := false
		if foundSvc.Spec.Type != app.Spec.ServiceType {
			foundSvc.Spec.Type = app.Spec.ServiceType
			needsUpdate = true
		}
		if foundSvc.Spec.Ports[0].Port != app.Spec.Port {
			foundSvc.Spec.Ports[0].Port = app.Spec.Port
			needsUpdate = true
		}
		if needsUpdate {
			log.Info("更新 Service", "name", foundSvc.Name)
			if err := r.Update(ctx, foundSvc); err != nil {
				log.Error(err, "更新 Service 失败")
				return ctrl.Result{}, err
			}
		}
	}

	// 4. 更新 SimpleApp 状态
	if err := r.Get(ctx, types.NamespacedName{Name: deploy.Name, Namespace: deploy.Namespace}, foundDeploy); err == nil {
		app.Status.Replicas = *foundDeploy.Spec.Replicas
		app.Status.Ready = foundDeploy.Status.ReadyReplicas
		if err := r.Status().Update(ctx, app); err != nil {
			log.Error(err, "更新 SimpleApp 状态失败")
			return ctrl.Result{}, err
		}
	}

	log.Info("Reconciliation 完成", "name", app.Name, "replicas", app.Status.Replicas, "ready", app.Status.Ready)
	return ctrl.Result{}, nil
}

// createDeployment 创建 Deployment（移除资源配额，避免编译错误）
func (r *SimpleAppReconciler) createDeployment(app *appv1.SimpleApp) *appsv1.Deployment {
	labels := map[string]string{"app": app.Name}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: app.Spec.Replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "app",
						Image: app.Spec.Image,
						Ports: []corev1.ContainerPort{{
							ContainerPort: app.Spec.Port,
							Protocol:      corev1.ProtocolTCP,
						}},
						// 移除 Resources 配置，避免 resource.MustParseQuantity 错误
						LivenessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/",
									Port: intstr.FromInt32(app.Spec.Port),
								},
							},
							InitialDelaySeconds: 30,
							PeriodSeconds:       10,
						},
						ReadinessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/",
									Port: intstr.FromInt32(app.Spec.Port),
								},
							},
							InitialDelaySeconds: 5,
							PeriodSeconds:       5,
						},
					}},
				},
			},
		},
	}
}

// createService 创建 Service
func (r *SimpleAppReconciler) createService(app *appv1.SimpleApp) *corev1.Service {
	labels := map[string]string{"app": app.Name}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type:     app.Spec.ServiceType,
			Selector: labels,
			Ports: []corev1.ServicePort{{
				Port:     app.Spec.Port,
				Protocol: corev1.ProtocolTCP,
			}},
		},
	}
}

// SetupWithManager 注册控制器
func (r *SimpleAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1.SimpleApp{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
