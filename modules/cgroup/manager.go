// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package cgroup

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"

	"forgejo.org/modules/container"
	"forgejo.org/modules/log"
)

const (
	// cgroup root files
	typeFile           = "cgroup.type"
	procsFile          = "cgroup.procs"
	controllersFile    = "cgroup.controllers"
	subtreeControlFile = "cgroup.subtree_control"

	// cgroup types
	domainType = "domain"

	// pid controller files
	pidsMaxFile     = "pids.max"
	pidsCurrentFile = "pids.current"
	pidsPeakFile    = "pids.peak"
)

type Manager struct {
	path                 string                // Absoloute path to the cgroup the manager is responsible for
	controllersAvailible container.Set[string] // Enabled cgroup controllers
}

func newManager(path string) (*Manager, error) {
	log.Trace("Created cgroup manager for %s", path)
	m := &Manager{
		path:                 path,
		controllersAvailible: nil,
	}

	// make sure the new cgroup is also of type domain
	cgroupType, err := m.ReadType()
	if err != nil {
		return nil, err
	}

	if cgroupType != domainType {
		return nil, ErrCgroupNotDomainType
	}

	controllers, err := m.ReadControllers()
	if err != nil {
		return nil, err
	}

	m.controllersAvailible = controllers

	return m, nil
}

// Create a new cgroup and coresponding manager. The common ancestor of the new
// group and group Forgejo is running in must be writable by the current
// process.
func NewManager(groupName string) (*Manager, error) {
	currentCgroup, err := ReadCurrentCgroup()
	if err != nil {
		return nil, err
	}

	// Loading the parent manager also makes sure it is of type domain
	_, err = LoadManager(currentCgroup)
	if err != nil {
		return nil, err
	}

	// create our now cgroup
	fullCgroupPath := path.Join(currentCgroup, groupName)
	err = os.MkdirAll(fullCgroupPath, 0o700)
	if err != nil {
		return nil, err
	}

	m, err := newManager(fullCgroupPath)
	return m, err
}

// Create a manager for an existing cgroup.
func LoadManager(cgroup string) (*Manager, error) {
	fileInfo, err := os.Stat(cgroup)
	if err != nil {
		return nil, err
	}

	if !fileInfo.IsDir() {
		return nil, ErrCgroupNotExists
	}

	return newManager(cgroup)
}

func (m *Manager) readFile(file string) (string, error) {
	filePath := path.Join(m.path, file)
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	trimmed := strings.TrimSuffix(string(raw), "\n")
	log.Trace("Read %s: %s", file, trimmed)
	return trimmed, nil
}

func (m *Manager) writeFile(file, content string) error {
	filePath := path.Join(m.path, file)
	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		return err
	}

	_, err = f.WriteString(content)
	return err
}

func (m *Manager) ReadType() (string, error) {
	cgroupType, err := m.readFile(typeFile)
	if err != nil {
		return "", err
	}

	return cgroupType, nil
}

func (m *Manager) ReadControllers() (container.Set[string], error) {
	controllersStr, err := m.readFile(controllersFile)
	if err != nil {
		return nil, err
	}

	controllerList := strings.Split(controllersStr, " ")
	controllerSet := container.SetOf(controllerList...)
	return controllerSet, nil
}

func (m *Manager) SetSubtreeControllers(newControllers container.Set[string]) error {
	isSubset := m.controllersAvailible.IsSubset(newControllers.Slice())
	if !isSubset {
		missing := newControllers.Difference(m.controllersAvailible).Slice()
		return ErrMissingControllers{missing}
	}

	var builder strings.Builder
	for _, c := range newControllers.Slice() {
		if builder.Len() > 0 {
			builder.WriteString(" ")
		}

		builder.WriteString("+")
		builder.WriteString(c)
	}

	controllersUpdate := builder.String()
	log.Trace("Enabling controllers: %s", controllersUpdate)
	return m.writeFile(subtreeControlFile, controllersUpdate)
}

func (m *Manager) AddProc(pid int) error {
	formattedProc := fmt.Sprintf("%d", pid)
	log.Trace("Adding process: %s", formattedProc)
	return m.writeFile(procsFile, formattedProc)
}

func (m *Manager) PidSetMax(amount string) error {
	log.Trace("Setting max PID amount: %s", amount)
	return m.writeFile(pidsMaxFile, amount)
}

func (m *Manager) PidGetMax() (string, error) {
	pidsMax, err := m.readFile(pidsMaxFile)
	if err != nil {
		return "", nil
	}

	return pidsMax, nil
}

func (m *Manager) PidGetCurrent() (uint64, error) {
	pidsCurrent, err := m.readFile(pidsCurrentFile)
	if err != nil {
		return 0, err
	}

	return strconv.ParseUint(pidsCurrent, 10, 64)
}

func (m *Manager) PidGetPeak() (uint64, error) {
	pidsCurrent, err := m.readFile(pidsPeakFile)
	if err != nil {
		return 0, err
	}

	return strconv.ParseUint(pidsCurrent, 10, 64)
}

func (m *Manager) StartAndMove(cmd *exec.Cmd) error {
	maxPids, err := m.PidGetMax()
	if err != nil {
		return err
	}

	if maxPids != "max" {
		maxPids, err := strconv.ParseUint(maxPids, 10, 64)
		if err != nil {
			return err
		}

		currentPids, err := m.PidGetCurrent()
		if err != nil {
			return err
		}

		if currentPids >= maxPids {
			return errors.New("Cgroup has no more free PIDs")
		}
	}

	err = cmd.Start()
	if err != nil {
		return err
	}

	return m.AddProc(cmd.Process.Pid)
}
