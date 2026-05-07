<script setup>
import Modal from "./Modal.vue";
import { peerStore } from "@/stores/peers";
import { interfaceStore } from "@/stores/interfaces";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { freshInterface, freshPeer, freshStats } from '@/helpers/models';
import Prism from "vue-prism-component";
import { notify } from "@kyvg/vue3-notification";
import { settingsStore } from "@/stores/settings";
import { profileStore } from "@/stores/profile";
import { base64_url_encode } from '@/helpers/encoding';
import { apiWrapper } from "@/helpers/fetch-wrapper";

const { t } = useI18n()

const settings = settingsStore()
const peers = peerStore()
const interfaces = interfaceStore()
const profile = profileStore()

const props = defineProps({
  peerId: String,
  visible: Boolean,
})

const emit = defineEmits(['close'])

function close() {
  emit('close')
}

const configString = ref("")

const selectedPeer = computed(() => {
  let p = peers.Find(props.peerId)

  if (!p) {
    if (!!props.peerId || props.peerId.length) {
      p = profile.peers.find((p) => p.Identifier === props.peerId)
    } else {
      p = freshPeer() // dummy peer to avoid 'undefined' exceptions
    }
  }
  return p
})

const selectedStats = computed(() => {
  let s = peers.Statistics(props.peerId)

  if (!s) {
    if (!!props.peerId || props.peerId.length) {
      s = profile.Statistics(props.peerId)
    } else {
      s = freshStats() // dummy stats to avoid 'undefined' exceptions
    }

  }
  return s
})

const selectedInterface = computed(() => {
  let i = interfaces.GetSelected;

  if (!i) {
    i = freshInterface() // dummy interface to avoid 'undefined' exceptions
  }
  return i
})

const title = computed(() => {
  if (!props.visible) {
    return "" // otherwise interfaces.GetSelected will die...
  }
  if (selectedInterface.value.Mode === "server") {
    return t("modals.peer-view.headline-peer") + " " + selectedPeer.value.DisplayName
  } else {
    return t("modals.peer-view.headline-endpoint") + " " + selectedPeer.value.DisplayName
  }
})

const configStyle = ref("wgquick")

// Client-app helper for the QR-code download flow. Maps the server-side
// interface backend → the canonical client app the user needs to import
// the .conf into. AmneziaWG-backed interfaces require the AmneziaVPN
// client (vanilla WireGuard apps cannot complete the handshake because
// of the obfuscated H1-H4 magic bytes); plain WG-backed interfaces use
// the standard WireGuard client. A null return hides the inline link.
const clientAppLink = computed(() => {
  if (selectedInterface.value.Mode === 'client') return null
  if (selectedInterface.value.Backend === 'amneziawg') {
    return { name: 'AmneziaVPN', url: 'https://amnezia.org/downloads' }
  }
  return { name: 'WireGuard', url: 'https://www.wireguard.com/install/' }
})

// 'clean' strips wg-portal's `# -WGP-` metadata comments + blank lines so the
// displayed/copied config matches what a user would type by hand. 'full'
// shows everything the API returns (with metadata comments).
const configView = ref('full')

const displayedConfig = computed(() => {
  if (!configString.value) return ''
  if (configView.value === 'full') return configString.value
  return configString.value
    .split('\n')
    .filter(line => !line.trimStart().startsWith('#'))
    .filter((line, idx, arr) => !(line === '' && (idx === 0 || arr[idx - 1] === '')))
    .join('\n')
})

async function copyConfig() {
  try {
    await navigator.clipboard.writeText(displayedConfig.value)
    notify({ title: t('modals.peer-view.copy-success'), type: 'success' })
  } catch (e) {
    notify({ title: t('modals.peer-view.copy-failed'), text: e.toString(), type: 'error' })
  }
}

// Renders the QR <img> onto a canvas and triggers a PNG download. Avoids
// a server round-trip — same QR the modal already shows. Falls back
// gracefully if the image hasn't loaded yet (img.naturalWidth=0).
function downloadQrPng() {
  const img = document.querySelector('.config-qr-img')
  if (!img || !img.naturalWidth) {
    notify({ title: t('modals.peer-view.qr-download-failed'), type: 'warn' })
    return
  }
  const canvas = document.createElement('canvas')
  canvas.width = img.naturalWidth
  canvas.height = img.naturalHeight
  canvas.getContext('2d').drawImage(img, 0, 0)
  const a = document.createElement('a')
  a.href = canvas.toDataURL('image/png')
  a.download = (selectedPeer.value.Filename || 'peer').replace(/\.conf$/, '') + '-qr.png'
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
}

watch(() => props.visible, async (newValue, oldValue) => {
  if (oldValue === false && newValue === true) { // if modal is shown
    // Default the style toggle to match the interface backend so the
    // AmneziaWG option is preselected for AWG peers — matches the
    // server-side auto-promote in configfile/manager.go and makes the
    // UI honest about what the rendered .conf actually contains.
    if (selectedInterface.value.Backend === 'amneziawg' && configStyle.value === 'wgquick') {
      configStyle.value = 'amneziawg'
    } else if (selectedInterface.value.Backend !== 'amneziawg' && configStyle.value === 'amneziawg') {
      configStyle.value = 'wgquick'
    }
    await peers.LoadPeerConfig(selectedPeer.value.Identifier, configStyle.value)
    configString.value = peers.configuration
  }
})

watch(() => configStyle.value, async () => {
  await peers.LoadPeerConfig(selectedPeer.value.Identifier, configStyle.value)
  configString.value = peers.configuration
})

function download() {
  // credit: https://www.bitdegree.org/learn/javascript-download
  let text = configString.value

  let element = document.createElement('a')
  element.setAttribute('href', 'data:application/octet-stream;charset=utf-8,' + encodeURIComponent(text))
  element.setAttribute('download', selectedPeer.value.Filename)

  element.style.display = 'none'
  document.body.appendChild(element)

  element.click()
  document.body.removeChild(element)
}

function email() {
  peers.MailPeerConfig(settings.Setting("MailLinkOnly"), configStyle.value, [selectedPeer.value.Identifier]).catch(e => {
    notify({
      title: "Failed to send mail with peer configuration!",
      text: e.toString(),
      type: 'error',
    })
  })
}

function ConfigQrUrl() {
  if (props.peerId.length) {
    return apiWrapper.url(`/peer/config-qr/${base64_url_encode(props.peerId)}?style=${configStyle.value}`)
  }
  return ''
}

</script>

<template>
  <Modal :title="title" :visible="visible" @close="close">
    <template #default>
      <div class="d-flex justify-content-end align-items-center mb-1" v-if="selectedInterface.Mode !== 'client'">
        <span class="me-2">{{ $t('modals.peer-view.style-label') }}: </span>
        <div class="btn-group btn-switch-group" role="group" aria-label="Configuration Style">
          <input type="radio" class="btn-check" name="configstyle" id="raw" value="raw" autocomplete="off" v-model="configStyle">
          <label class="btn btn-outline-dark btn-sm" for="raw">Raw</label>
          <input type="radio" class="btn-check" name="configstyle" id="wgquick" value="wgquick" autocomplete="off" v-model="configStyle">
          <label class="btn btn-outline-dark btn-sm" for="wgquick">WG-Quick</label>
          <!-- AmneziaWG style — only meaningful for AWG-backed interfaces.
               Backend defaults to 'local' for legacy interfaces, so we
               render this option whenever the backend is explicitly
               'amneziawg'. Picking it embeds the obfuscation params
               (Jc/Jmin/Jmax/S1-S4/H1-H4/I1-I5) in the rendered .conf. -->
          <template v-if="selectedInterface.Backend === 'amneziawg'">
            <input type="radio" class="btn-check" name="configstyle" id="amneziawg" value="amneziawg" autocomplete="off" v-model="configStyle">
            <label class="btn btn-outline-dark btn-sm" for="amneziawg">AmneziaWG</label>
          </template>
        </div>
      </div>
      <div class="accordion" id="peerInformation">
        <div class="accordion-item">
          <h2 class="accordion-header">
            <button class="accordion-button" type="button" data-bs-toggle="collapse" data-bs-target="#collapseDetails"
              aria-expanded="true" aria-controls="collapseDetails">
              {{ $t('modals.peer-view.section-info') }}
            </button>
          </h2>
          <div id="collapseDetails" class="accordion-collapse collapse show" aria-labelledby="headingDetails"
            data-bs-parent="#peerInformation" style="">
            <div class="accordion-body">
              <!-- Peer details (full width) — list of identifiers, IPs, etc. -->
              <ul>
                <li v-if="selectedInterface.Mode !== 'client'"><strong>{{ $t('modals.peer-view.identifier') }}</strong>: {{ selectedPeer.PublicKey }}</li>
                <li v-if="selectedInterface.Mode !== 'server'"><strong>{{ $t('modals.peer-view.endpoint-key') }}</strong>: {{ selectedPeer.PublicKey }}</li>
                <li v-if="selectedInterface.Mode !== 'server'"><strong>{{ $t('modals.peer-view.endpoint') }}</strong>: {{ selectedPeer.Endpoint.Value }}</li>
                <li v-if="selectedInterface.Mode !== 'client'"><strong>{{ $t('modals.peer-view.ip') }}</strong>: <span v-for="ip in selectedPeer.Addresses" :key="ip"
                    class="badge rounded-pill bg-light">{{ ip }}</span></li>
                <li v-if="selectedInterface.Mode === 'server'"><strong>{{ $t('modals.peer-view.extra-allowed-ip') }}</strong>: <span v-for="ip in selectedPeer.ExtraAllowedIPs" :key="ip"
                                                                                                                    class="badge rounded-pill bg-light">{{ ip }}</span></li>
                <li v-if="selectedInterface.Mode !== 'server' && selectedPeer.AllowedIPs.Value"><strong>{{ $t('modals.peer-view.allowed-ip') }}</strong>: <span v-for="ip in selectedPeer.AllowedIPs.Value" :key="ip"
                                                                                                      class="badge rounded-pill bg-light">{{ ip }}</span></li>
                <li v-if="selectedInterface.Mode !== 'server'"><strong>{{ $t('modals.peer-view.keepalive') }}</strong>: {{ selectedPeer.PersistentKeepalive.Value }}</li>
                <li v-if="selectedPeer.UserDisplayName"><strong>{{ $t('modals.peer-view.user') }}</strong>: {{ selectedPeer.UserDisplayName }} ({{ selectedPeer.UserIdentifier }})</li>
                <li v-else><strong>{{ $t('modals.peer-view.user') }}</strong>: {{ selectedPeer.UserIdentifier }}</li>
                <li v-if="selectedPeer.Notes"><strong>{{ $t('modals.peer-view.notes') }}</strong>: {{ selectedPeer.Notes }}</li>
                <li v-if="selectedPeer.ExpiresAt"><strong>{{ $t('modals.peer-view.expiry-status') }}</strong>: {{
                  selectedPeer.ExpiresAt }}</li>
                <li v-if="selectedPeer.Disabled"><strong>{{ $t('modals.peer-view.disabled-status') }}</strong>: {{
                  selectedPeer.DisabledReason }}</li>
              </ul>

              <!-- QR + config side-by-side panel (server-mode peers only) -->
              <div v-if="selectedInterface.Mode !== 'client'" class="row mt-3">
                <!-- QR column -->
                <div class="col-lg-5 d-flex flex-column align-items-center mb-3">
                  <img class="config-qr-img" :src="ConfigQrUrl()" loading="lazy" alt="Configuration QR Code">
                  <button type="button" class="btn btn-outline-secondary btn-sm mt-2" @click.prevent="downloadQrPng">
                    <i class="fas fa-download me-1"></i>{{ $t('modals.peer-view.button-download-qr') }}
                  </button>
                  <p v-if="clientAppLink" class="small text-muted mt-2 mb-0 text-center">
                    {{ $t('modals.peer-view.client-app-prompt') }}
                    <a :href="clientAppLink.url" target="_blank" rel="noopener noreferrer">{{ clientAppLink.name }}</a>
                  </p>
                </div>
                <!-- Config text column -->
                <div class="col-lg-7">
                  <div class="d-flex align-items-center mb-2">
                    <strong class="me-2">{{ $t('modals.peer-view.config-label') }}</strong>
                    <div class="btn-group btn-switch-group ms-auto me-2" role="group" aria-label="Config view">
                      <input type="radio" class="btn-check" id="cfg-clean" value="clean" v-model="configView" autocomplete="off">
                      <label class="btn btn-outline-dark btn-sm" for="cfg-clean">{{ $t('modals.peer-view.config-clean') }}</label>
                      <input type="radio" class="btn-check" id="cfg-full" value="full" v-model="configView" autocomplete="off">
                      <label class="btn btn-outline-dark btn-sm" for="cfg-full">{{ $t('modals.peer-view.config-full') }}</label>
                    </div>
                    <button type="button" class="btn btn-outline-secondary btn-sm" :title="$t('modals.peer-view.button-copy')" @click.prevent="copyConfig">
                      <i class="fas fa-clipboard"></i>
                    </button>
                  </div>
                  <Prism language="ini" :code="displayedConfig" class="config-side-by-side"></Prism>
                </div>
              </div>
            </div>
          </div>
        </div>
        <div class="accordion-item">
          <h2 class="accordion-header" id="headingStatus">
            <button class="accordion-button collapsed" type="button" data-bs-toggle="collapse"
              data-bs-target="#collapseStatus" aria-expanded="false" aria-controls="collapseStatus">
              {{ $t('modals.peer-view.section-status') }}
            </button>
          </h2>
          <div id="collapseStatus" class="accordion-collapse collapse" aria-labelledby="headingStatus"
            data-bs-parent="#peerInformation" style="">
            <div class="accordion-body">
              <div class="row">
                <div class="col-md-12">
                  <h4>{{ $t('modals.peer-view.traffic') }}</h4>
                  <p><i class="fas fa-long-arrow-alt-down" :title="$t('modals.peer-view.download')"></i> {{
                    selectedStats.BytesReceived }} Bytes / <i class="fas fa-long-arrow-alt-up"
                      :title="$t('modals.peer-view.upload')"></i> {{ selectedStats.BytesTransmitted }} Bytes</p>
                  <h4>{{ $t('modals.peer-view.connection-status') }}</h4>
                  <ul>
                    <li>{{ $t('modals.peer-view.pingable') }}: {{ selectedStats.IsPingable }}</li>
                    <li>{{ $t('modals.peer-view.handshake') }}: {{ selectedStats.LastHandshake }}</li>
                    <li>{{ $t('modals.peer-view.connected-since') }}: {{ selectedStats.LastSessionStart }}</li>
                    <li>{{ $t('modals.peer-view.endpoint') }}: {{ selectedStats.EndpointAddress }}</li>
                  </ul>
                </div>
              </div>
            </div>
          </div>
        </div>
        <!-- The legacy 'Configuration' accordion section is removed. The
             config text now lives next to the QR in the Information
             section above, with a Clean/Full toggle and a Copy button. -->
      </div>
    </template>
    <template #footer>
      <div class="flex-fill text-start">
        <button v-if="selectedInterface.Mode !== 'client'" @click.prevent="download" type="button" class="btn btn-primary me-1">{{
          $t('modals.peer-view.button-download') }}</button>
        <button v-if="selectedInterface.Mode !== 'client'" @click.prevent="email" type="button" class="btn btn-primary me-1">{{
          $t('modals.peer-view.button-email') }}</button>
      </div>
      <button @click.prevent="close" type="button" class="btn btn-secondary">{{ $t('general.close') }}</button>


  </template>
</Modal></template>

<style>
.config-qr-img {
  max-width: 100%;
}

.btn-switch-group .btn {
  border-width: 1px;
  padding: 5px;
  line-height: 1;
}

/* Side-by-side config panel: keep the code block from blowing the
   modal out when the config is long (esp. AmneziaWG with I1-I5). */
.config-side-by-side {
  max-height: 22rem;
  overflow: auto;
  font-size: 0.85rem;
  margin: 0;
}
.config-side-by-side pre {
  margin: 0;
}
</style>
