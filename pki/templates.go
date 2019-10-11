package pki

import (
	"os"
	"path/filepath"

	"github.com/smallstep/cli/utils"

	"github.com/pkg/errors"
	"github.com/smallstep/certificates/templates"
	"github.com/smallstep/cli/errs"
)

// sshTemplates contains the configuration of default templates used on ssh.
var sshTemplates = &templates.SSHTemplates{
	User: []templates.Template{
		{Name: "include.tpl", Type: templates.Snippet, TemplatePath: "ssh/include.tpl", Path: "~/.ssh/config", Comment: "#"},
		{Name: "config.tpl", Type: templates.File, TemplatePath: "ssh/config.tpl", Path: "ssh/config", Comment: "#"},
		{Name: "known_hosts.tpl", Type: templates.File, TemplatePath: "ssh/known_hosts.tpl", Path: "ssh/known_hosts", Comment: "#"},
	},
	Host: []templates.Template{
		{Name: "sshd_config.tpl", Type: templates.Snippet, TemplatePath: "ssh/sshd_config.tpl", Path: "/etc/ssh/sshd_config", Comment: "#"},
		{Name: "ca.tpl", Type: templates.Snippet, TemplatePath: "ssh/ca.tpl", Path: "/etc/ssh/ca.pub", Comment: "#"},
	},
}

// sshTemplateData contains the data of the default templates used on ssh.
var sshTemplateData = map[string]string{
	// include.tpl adds the step ssh config file
	"include.tpl": `Host *
	Include {{.User.StepPath}}/ssh/config`,

	// config.tpl is the step ssh config file, it includes the Match rule
	// and references the step known_hosts file
	"config.tpl": `Match exec "step ssh check-host %h"
	ForwardAgent yes
	UserKnownHostsFile {{.User.StepPath}}/config/ssh/known_hosts`,

	// known_hosts.tpl authorizes the ssh hosst key
	"known_hosts.tpl": "@cert-authority * {{.Step.SSH.HostKey.Type}} {{.Step.SSH.HostKey.Marshal | toString | b64enc}}",

	// sshd_config.tpl adds the configuration to support certificates
	"sshd_config.tpl": `TrustedUserCAKeys /etc/ssh/ca.pub
HostCertificate /etc/ssh/{{.User.Certificate}}
HostKey /etc/ssh/{{.User.Key}}`,

	// ca.tpl contains the public key used to authorized clients
	"ca.tpl": "{{.Step.SSH.UserKey.Type}} {{.Step.SSH.UserKey.Marshal | toString | b64enc}}",
}

// getTemplates returns all the templates enabled
func (p *PKI) getTemplates() *templates.Templates {
	if !p.enableSSH {
		return nil
	}

	return &templates.Templates{
		SSH:  sshTemplates,
		Data: map[string]interface{}{},
	}
}

// generateTemplates generates given templates.
func generateTemplates(t *templates.Templates) error {
	if t == nil {
		return nil
	}

	base := GetTemplatesPath()
	// Generate SSH templates
	if t.SSH != nil {
		// all ssh templates are under ssh:
		sshDir := filepath.Join(base, "ssh")
		if _, err := os.Stat(sshDir); os.IsNotExist(err) {
			if err = os.MkdirAll(sshDir, 0700); err != nil {
				return errs.FileError(err, sshDir)
			}
		}
		// Create all templates
		for _, t := range t.SSH.User {
			data, ok := sshTemplateData[t.Name]
			if !ok {
				return errors.Errorf("template %s does not exists", t.Name)
			}
			if err := utils.WriteFile(filepath.Join(base, t.TemplatePath), []byte(data), 0644); err != nil {
				return err
			}
		}
		for _, t := range t.SSH.Host {
			data, ok := sshTemplateData[t.Name]
			if !ok {
				return errors.Errorf("template %s does not exists", t.Name)
			}
			if err := utils.WriteFile(filepath.Join(base, t.TemplatePath), []byte(data), 0644); err != nil {
				return err
			}
		}
	}

	return nil
}
