package glib

import (
	"fmt"
	"github.com/chwjbn/live-hub/glog"
	"github.com/kardianos/service"
	"os"
	"os/signal"
	"syscall"
)

type ServTaskFunc func() error

type Serv struct {
	mSrvName     string
	mSrvTitle    string
	mSrvInfo     string
	mSrvTaskFunc ServTaskFunc

	mService    service.Service
	mNotifyChan chan os.Signal
}

func NewServ(srvName string, srvTitle string, srvInfo string, srvTaskFunc ServTaskFunc) (*Serv, error) {

	srv := new(Serv)
	srv.mSrvName = srvName
	srvTitle = srvTitle
	srvInfo = srvInfo
	srv.mSrvTaskFunc = srvTaskFunc

	xErr := srv.init()
	if xErr != nil {
		return nil, xErr
	}

	return srv, xErr

}

func (this *Serv) init() error {

	var xErr error

	this.mNotifyChan = make(chan os.Signal)

	var srvErr error

	this.mService, srvErr = service.New(this, &service.Config{
		Name:        this.mSrvName,
		DisplayName: this.mSrvTitle,
		Description: this.mSrvInfo,
	})

	if srvErr != nil {
		xErr = fmt.Errorf("service create with error:[%s]", srvErr.Error())
	}

	return xErr

}

func (this *Serv) run() error {

	var xErr error

	go func() {
		if this.mSrvTaskFunc != nil {
			this.mSrvTaskFunc()
		}
	}()

	signal.Notify(this.mNotifyChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	for {
		select {
		case xSignal := <-this.mNotifyChan:

			glog.Info(fmt.Sprintf("Serv Notify Exit Signal=[%v]", xSignal.String()))
			os.Exit(0)

		}
	}

	return xErr

}

func (this *Serv) Start(s service.Service) error {

	var xErr error

	glog.InfoF("Serv.Start Name=[%s] begin", this.mSrvName)

	go func() {
		this.run()
	}()

	glog.InfoF("Serv.Start Name=[%s] end", this.mSrvName)

	return xErr
}

func (this *Serv) Stop(s service.Service) error {

	var xErr error

	glog.InfoF("Serv.Stop Name=[%s] begin", this.mSrvName)

	this.mNotifyChan <- syscall.SIGQUIT

	glog.InfoF("Serv.Stop Name=[%s] end", this.mSrvName)

	return xErr

}

func (this *Serv) Install() {

	glog.Info("Serv.Install begin")

	this.mService.Install()

	glog.Info("Serv.Install end")

}

func (this *Serv) Uninstall() {

	glog.Info("Serv.Uninstall begin")

	this.mService.Uninstall()

	glog.Info("Serv.Uninstall end")

}

func (this *Serv) RunService() {

	glog.Info("Serv.RunService begin")

	this.mService.Run()

	glog.Info("Serv.RunService end")

}

func (this *Serv) StartService() {

	glog.Info("Serv.StartService begin")

	this.mService.Start()

	glog.Info("Serv.StartService end")
}

func (this *Serv) StopService() {

	glog.Info("Serv.StopService begin")

	this.mService.Stop()

	glog.Info("Serv.StopService end")

}
