# frozen_string_literal: true

require 'rspec'
require 'bosh/template/test'
require_relative 'spec_helper'

describe 'loggr-syslog-agent job' do
  let(:release_dir) { File.join(File.dirname(__FILE__), '..') }
  let(:release) { Bosh::Template::Test::ReleaseDir.new(release_dir) }
  let(:job) { release.job('loggr-syslog-agent') }

  describe 'drain_ca.cert' do
    let(:template) { job.template('config/certs/drain_ca.crt') }

    it 'can render the template' do
      properties = {
        'drain_ca_cert' => 'drain_ca_cert'
      }

      actual = template.render(properties)
      expect(actual).to match('drain_ca_cert')
    end
  end

  describe 'bpm.yml' do
    let(:template) { job.template('config/bpm.yml') }
    
    it 'contains the tls config for the syslog drain to log-cache' do
      certPath = "/var/vcap/jobs/loggr-syslog-agent/config/certs"
      properties = {
        'drain_ca_cert' => "#{certPath}/drain_ca.crt",
        'drain_use_mtls' => "true",
      }
      bpm_yml = YAML.safe_load(template.render(properties))
      env = bpm_process(bpm_yml, 0)['env']
      expect(env).to include('DRAIN_TRUSTED_CA_FILE')
      expect(env).to include('DRAIN_USE_MTLS')
      expect(env.fetch("DRAIN_TRUSTED_CA_FILE")).to eq("#{certPath}/drain_ca.crt")
      expect(env.fetch("DRAIN_USE_MTLS")).to eq("true")
    end
  end
end
