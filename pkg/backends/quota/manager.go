/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

//Package quota is a alpha feature. it manage service quota
//quota management can not assure you strong consistency
package quota

import (
	"errors"
	"fmt"
	"github.com/go-chassis/openlog"
)

//errors
var (
	ErrReached   = errors.New("reached maximum allowed quota")
	ErrGetFailed = errors.New("get quota failed")
)

type newManager func(opts Options) (Manager, error)

var plugins = make(map[string]newManager)

//Install install quota plugin
func Install(name string, f newManager) {
	plugins[name] = f
}

//Init init manager
func Init(opts Options) error {
	if opts.Plugin == "" {
		return nil
	}

	f, ok := plugins[opts.Plugin]
	if !ok {
		return fmt.Errorf("not supported [%s]", opts.Plugin)
	}
	var err error
	defaultManager, err = f(opts)
	if err != nil {
		return err
	}
	openlog.Info(fmt.Sprintf("quota management system [%s@%s] enabled", opts.Plugin, opts.Endpoint))
	return nil
}

//defaultManager is manage quotas
var defaultManager Manager

// Quota describe quota infos
type Quota struct {
	ResourceType string
	Limit        int64
	Used         int64
	Unit         string
}

//Manager could be a quota management system as a remote service, which saves and manages all of your system resources.
// or it could be a module of your service which manage quota saved in database
type Manager interface {
	GetQuota(domain, project, resourceType string) (*Quota, error)
	GetQuotas(domain, project string) ([]*Quota, error)
	IncreaseUsed(domain, project, resourceType string, used int64) error
	DecreaseUsed(domain, project, resourceType string, used int64) error
	SetLimit(domain, project, resourceType string, limit int64) error
}

//PreCreate only check quota usage before creating a resource for a domain(tenant) and project.
//it will not increase resource usage number after check, you have to increase after resource actually created
func PreCreate(domain, project, resource string, number int64) error {
	if defaultManager == nil {
		openlog.Warn("quota management not available, fallback")
		return nil
	}
	qs, err := defaultManager.GetQuotas(domain, project)
	if err != nil {
		openlog.Error(err.Error())
		return ErrGetFailed
	}
	var resourceQuota *Quota
	for _, q := range qs {
		if q.ResourceType == resource {
			resourceQuota = q
			break
		}
	}
	if resourceQuota == nil {
		//no limits
		openlog.Debug("no limits for " + resource)
		return nil
	}
	if number > resourceQuota.Limit-resourceQuota.Used {
		return ErrReached
	}
	return nil
}
