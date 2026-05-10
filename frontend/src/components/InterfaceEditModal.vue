<script setup>
import Modal from "./Modal.vue";
import {interfaceStore} from "@/stores/interfaces";
import {computed, ref, watch} from "vue";
import { useI18n } from 'vue-i18n';
import { notify } from "@kyvg/vue3-notification";
import { VueTagsInput } from '@vojtechlanka/vue-tags-input';
import { validateCIDR, validateIP, validateDomain } from '@/helpers/validators';
import isCidr from "is-cidr";
import {isIP} from 'is-ip';
import { freshInterface, freshAmneziaWG, freshUserPool } from '@/helpers/models';
import { cidrError, cidrContains } from '@/helpers/cidr';
import {peerStore} from "@/stores/peers";
import {settingsStore} from "@/stores/settings";

const { t } = useI18n()

const interfaces = interfaceStore()
const peers = peerStore()
const settings = settingsStore()

const props = defineProps({
  interfaceId: String,
  visible: Boolean,
})

const emit = defineEmits(['close'])

const selectedInterface = computed(() => {
  return interfaces.Find(props.interfaceId)
})

const title = computed(() => {
  if (!props.visible) {
    return "" // otherwise interfaces.GetSelected will die...
  }

  if (selectedInterface.value) {
    return t("modals.interface-edit.headline-edit") + " " + selectedInterface.value.Identifier
  }
  return t("modals.interface-edit.headline-new")
})

const currentTags = ref({
  Addresses: "",
  Dns: "",
  DnsSearch: "",
  PeerDefNetwork: "",
  PeerDefAllowedIPs: "",
  PeerDefDns: "",
  PeerDefDnsSearch: ""
})
const formData = ref(freshInterface())
const isSaving = ref(false)
const isDeleting = ref(false)
const isApplyingDefaults = ref(false)

const isBackendValid = computed(() => {
  if (!props.visible || !selectedInterface.value) {
    return true // if modal is not visible or no interface is selected, we don't care about backend validity
  }

  let backendId = selectedInterface.value.Backend

  let valid = false
  let availableBackends = settings.Setting('AvailableBackends') || []
  availableBackends.forEach(backend => {
    if (backend.Id === backendId) {
      valid = true
    }
  })
  return valid
})

// functions

watch(() => props.visible, async (newValue, oldValue) => {
      if (oldValue === false && newValue === true) { // if modal is shown
        console.log(selectedInterface.value)
        if (!selectedInterface.value) {
          await interfaces.PrepareInterface()

          // fill form data
          formData.value.Identifier = interfaces.Prepared.Identifier
          formData.value.DisplayName = interfaces.Prepared.DisplayName
          formData.value.Mode = interfaces.Prepared.Mode
          formData.value.CreateDefaultPeer = interfaces.Prepared.CreateDefaultPeer
          formData.value.Backend = interfaces.Prepared.Backend

          formData.value.PublicKey = interfaces.Prepared.PublicKey
          formData.value.PrivateKey = interfaces.Prepared.PrivateKey

          formData.value.ListenPort = interfaces.Prepared.ListenPort
          formData.value.Addresses = interfaces.Prepared.Addresses
          formData.value.Dns = interfaces.Prepared.Dns
          formData.value.DnsSearch = interfaces.Prepared.DnsSearch

          formData.value.Mtu = interfaces.Prepared.Mtu
          formData.value.FirewallMark = interfaces.Prepared.FirewallMark
          formData.value.RoutingTable = interfaces.Prepared.RoutingTable

          formData.value.PreUp = interfaces.Prepared.PreUp
          formData.value.PostUp = interfaces.Prepared.PostUp
          formData.value.PreDown = interfaces.Prepared.PreDown
          formData.value.PostDown = interfaces.Prepared.PostDown

          formData.value.SaveConfig = interfaces.Prepared.SaveConfig

          formData.value.PeerDefNetwork = interfaces.Prepared.PeerDefNetwork
          formData.value.PeerDefDns = interfaces.Prepared.PeerDefDns
          formData.value.PeerDefDnsSearch = interfaces.Prepared.PeerDefDnsSearch
          formData.value.PeerDefEndpoint = interfaces.Prepared.PeerDefEndpoint
          formData.value.PeerDefAllowedIPs = interfaces.Prepared.PeerDefAllowedIPs
          formData.value.PeerDefMtu = interfaces.Prepared.PeerDefMtu
          formData.value.PeerDefPersistentKeepalive = interfaces.Prepared.PeerDefPersistentKeepalive
          formData.value.PeerDefFirewallMark = interfaces.Prepared.PeerDefFirewallMark
          formData.value.PeerDefRoutingTable = interfaces.Prepared.PeerDefRoutingTable
          formData.value.PeerDefPreUp = interfaces.Prepared.PeerDefPreUp
          formData.value.PeerDefPostUp = interfaces.Prepared.PeerDefPostUp
          formData.value.PeerDefPreDown = interfaces.Prepared.PeerDefPreDown
          formData.value.PeerDefPostDown = interfaces.Prepared.PeerDefPostDown
          formData.value.AmneziaWG = interfaces.Prepared.AmneziaWG || freshAmneziaWG()
          formData.value.UserPool = interfaces.Prepared.UserPool || freshUserPool()
        } else { // fill existing userdata
          formData.value.Disabled = selectedInterface.value.Disabled
          formData.value.Identifier = selectedInterface.value.Identifier
          formData.value.DisplayName = selectedInterface.value.DisplayName
          formData.value.Mode = selectedInterface.value.Mode
          formData.value.CreateDefaultPeer = selectedInterface.value.CreateDefaultPeer
          formData.value.Backend = selectedInterface.value.Backend

          formData.value.PublicKey = selectedInterface.value.PublicKey
          formData.value.PrivateKey = selectedInterface.value.PrivateKey

          formData.value.ListenPort = selectedInterface.value.ListenPort
          formData.value.Addresses = selectedInterface.value.Addresses
          formData.value.Dns = selectedInterface.value.Dns
          formData.value.DnsSearch = selectedInterface.value.DnsSearch

          formData.value.Mtu = selectedInterface.value.Mtu
          formData.value.FirewallMark = selectedInterface.value.FirewallMark
          formData.value.RoutingTable = selectedInterface.value.RoutingTable

          formData.value.PreUp = selectedInterface.value.PreUp
          formData.value.PostUp = selectedInterface.value.PostUp
          formData.value.PreDown = selectedInterface.value.PreDown
          formData.value.PostDown = selectedInterface.value.PostDown

          formData.value.SaveConfig = selectedInterface.value.SaveConfig

          formData.value.PeerDefNetwork = selectedInterface.value.PeerDefNetwork
          formData.value.PeerDefDns = selectedInterface.value.PeerDefDns
          formData.value.PeerDefDnsSearch = selectedInterface.value.PeerDefDnsSearch
          formData.value.PeerDefEndpoint = selectedInterface.value.PeerDefEndpoint
          formData.value.PeerDefAllowedIPs = selectedInterface.value.PeerDefAllowedIPs
          formData.value.PeerDefMtu = selectedInterface.value.PeerDefMtu
          formData.value.PeerDefPersistentKeepalive = selectedInterface.value.PeerDefPersistentKeepalive
          formData.value.PeerDefFirewallMark = selectedInterface.value.PeerDefFirewallMark
          formData.value.PeerDefRoutingTable = selectedInterface.value.PeerDefRoutingTable
          formData.value.PeerDefPreUp = selectedInterface.value.PeerDefPreUp
          formData.value.PeerDefPostUp = selectedInterface.value.PeerDefPostUp
          formData.value.PeerDefPreDown = selectedInterface.value.PeerDefPreDown
          formData.value.PeerDefPostDown = selectedInterface.value.PeerDefPostDown
          // AmneziaWG params: server returns null for non-AWG backends; we
          // keep a fully-zeroed default so binding never fails. The
          // AWG-specific fieldset is hidden unless Backend === 'amneziawg'.
          formData.value.AmneziaWG = selectedInterface.value.AmneziaWG || freshAmneziaWG()
          // Per-(user × interface) pool config (BNet-m76e). Always
          // emitted by the API; default-empty for non-anycast interfaces.
          formData.value.UserPool = selectedInterface.value.UserPool || freshUserPool()
          // Load read-only allocator state for the panel.
          await interfaces.LoadPoolState(selectedInterface.value.Identifier)
        }
      }
    }
)

function close() {
  formData.value = freshInterface()
  emit('close')
}

// AmneziaWG parameter constraints. Recommended ranges from
// github.com/amnezia-vpn/amneziawg-go README + practical experience.
// Each entry: { min, max, recommended: [lo, hi], description }.
const awgParamSpec = {
  Jc: { min: 0, max: 128, recommended: [3, 10], description: 'Junk packet count (sent before each handshake)', type: 'int' },
  Jmin: { min: 0, max: 1280, recommended: [50, 100], description: 'Junk packet minimum size (bytes)', type: 'int' },
  Jmax: { min: 0, max: 1280, recommended: [200, 1000], description: 'Junk packet maximum size (bytes); must exceed Jmin', type: 'int' },
  S1: { min: 0, max: 150, recommended: [15, 150], description: 'Init packet stuffing (bytes prepended to handshake init); cannot be 0 if S2 is set', type: 'int' },
  S2: { min: 0, max: 150, recommended: [15, 150], description: 'Response packet stuffing (bytes prepended to handshake response)', type: 'int' },
  S3: { min: 0, max: 1280, recommended: [50, 200], description: 'V2 init padding (extra bytes inside handshake init); 0 = disabled', type: 'int' },
  S4: { min: 0, max: 1280, recommended: [100, 300], description: 'V2 response padding (extra bytes inside handshake response); 0 = disabled', type: 'int' },
  H1: { min: 5, max: 4294967295, recommended: [5, 1147483647], description: 'Init magic header (replaces vanilla WG type=1). Must differ from H2-H4 and be > 4', type: 'uint32' },
  H2: { min: 5, max: 4294967295, recommended: [5, 1147483647], description: 'Response magic header (replaces vanilla WG type=2). Must differ from H1, H3, H4', type: 'uint32' },
  H3: { min: 5, max: 4294967295, recommended: [5, 1147483647], description: 'Cookie magic header (replaces vanilla WG type=3). Must differ from H1, H2, H4', type: 'uint32' },
  H4: { min: 5, max: 4294967295, recommended: [5, 1147483647], description: 'Data magic header (replaces vanilla WG type=4). Must differ from H1-H3', type: 'uint32' },
  I1: { description: 'V2 packet-injection bytes (hex string). Empty = disabled.', type: 'hex' },
  I2: { description: 'V2 packet-injection bytes (hex string). Empty = disabled.', type: 'hex' },
  I3: { description: 'V2 packet-injection bytes (hex string). Empty = disabled.', type: 'hex' },
  I4: { description: 'V2 packet-injection bytes (hex string). Empty = disabled.', type: 'hex' },
  I5: { description: 'V2 packet-injection bytes (hex string). Empty = disabled.', type: 'hex' },
}

// Live validation of the AmneziaWG fieldset. Runs on every form-data
// change (computed, no manual trigger). Returns an object keyed by
// param ID → error message string, or null when the field is OK.
const awgValidationErrors = computed(() => {
  if (formData.value.Backend !== 'amneziawg') return {}
  const a = formData.value.AmneziaWG || {}
  const errors = {}

  // Jmin < Jmax sanity (Jmax may be 0 to disable junk; if both > 0, Jmin must be < Jmax)
  if (a.Jmax > 0 && a.Jmin >= a.Jmax) {
    errors.Jmin = errors.Jmax = 'Jmin must be < Jmax (or both 0 to disable junk packets)'
  }

  // S1 != 0 when S2 != 0 (enforced by Amnezia, S1=0 with S2>0 is rejected)
  if (a.S2 > 0 && a.S1 === 0) {
    errors.S1 = 'S1 must be > 0 when S2 is set'
  }

  // S1 + 56 != S2 — Amnezia recommends this to avoid collisions (56 bytes = WG init payload)
  if (a.S1 > 0 && a.S2 > 0 && a.S1 + 56 === a.S2) {
    errors.S1 = errors.S2 = 'S1 + 56 must not equal S2 (causes packet-size collision with WG init)'
  }

  // H1-H4 uniqueness + > 4 (1-4 reserved for vanilla WG message types)
  const hVals = [a.H1, a.H2, a.H3, a.H4]
  hVals.forEach((v, i) => {
    const k = `H${i + 1}`
    if (v !== 0 && v < 5) {
      errors[k] = `${k} must be > 4 (1-4 are reserved for vanilla WireGuard message types)`
    }
  })
  if (hVals.filter(v => v > 0).length > 0) {
    const seen = new Map()
    hVals.forEach((v, i) => {
      if (v === 0) return
      if (seen.has(v)) {
        const k = `H${i + 1}`
        errors[k] = `${k} duplicates ${seen.get(v)}; H1-H4 must all differ`
      } else {
        seen.set(v, `H${i + 1}`)
      }
    })
  }

  // I1-I5 must be valid hex strings (or empty)
  for (const k of ['I1', 'I2', 'I3', 'I4', 'I5']) {
    const v = (a[k] || '').trim()
    if (v && !/^[0-9a-fA-F]+$/.test(v)) {
      errors[k] = `${k} must be a hex string (0-9, a-f) or empty`
    } else if (v && v.length % 2 !== 0) {
      errors[k] = `${k} hex string must have an even number of characters (whole bytes)`
    }
  }

  return errors
})

const hasAwgValidationErrors = computed(() => {
  return Object.keys(awgValidationErrors.value).length > 0
})

// Cryptographically-strong randint helper using window.crypto. Falls back
// to Math.random for hex-byte generation only (where biased randomness is
// fine — not for crypto material). uint32 seeds use crypto.
function cryptoRandUint32() {
  const buf = new Uint32Array(1)
  window.crypto.getRandomValues(buf)
  return buf[0]
}
function randInRange(lo, hi) {
  return lo + (cryptoRandUint32() % (hi - lo + 1))
}

// Populate AmneziaWG fields with sensible random defaults. Picks values
// from the recommended range for each numeric param, generates four
// distinct uint32s > 4 for H1-H4 (so they don't collide with vanilla
// WG types 1-4), and leaves I1-I5 empty (V2 injection is opt-in).
// Existing values are overwritten — operator confirmation expected via
// the surrounding "are you sure"-style notify message.
function generateRandomAwgParams() {
  const a = freshAmneziaWG()

  a.Jc = randInRange(awgParamSpec.Jc.recommended[0], awgParamSpec.Jc.recommended[1])
  a.Jmin = randInRange(awgParamSpec.Jmin.recommended[0], awgParamSpec.Jmin.recommended[1])
  // Make sure Jmax > Jmin
  a.Jmax = randInRange(Math.max(awgParamSpec.Jmax.recommended[0], a.Jmin + 50), awgParamSpec.Jmax.recommended[1])

  a.S1 = randInRange(awgParamSpec.S1.recommended[0], awgParamSpec.S1.recommended[1])
  // Avoid S1 + 56 === S2 collision: regenerate S2 if it lands on the bad value
  do {
    a.S2 = randInRange(awgParamSpec.S2.recommended[0], awgParamSpec.S2.recommended[1])
  } while (a.S1 + 56 === a.S2)
  a.S3 = randInRange(awgParamSpec.S3.recommended[0], awgParamSpec.S3.recommended[1])
  a.S4 = randInRange(awgParamSpec.S4.recommended[0], awgParamSpec.S4.recommended[1])

  // H1-H4: four distinct uint32s, all > 4. Loop until we get 4 unique values.
  const hSet = new Set()
  while (hSet.size < 4) {
    const v = cryptoRandUint32()
    if (v > 4) hSet.add(v)
  }
  const [h1, h2, h3, h4] = [...hSet]
  a.H1 = h1; a.H2 = h2; a.H3 = h3; a.H4 = h4

  formData.value.AmneziaWG = a
  notify({
    title: t('modals.interface-edit.amneziawg-randomized-title'),
    text: t('modals.interface-edit.amneziawg-randomized-body'),
    type: 'success',
  })
}

function handleChangeAddresses(tags) {
  let validInput = true
  tags.forEach(tag => {
    if(isCidr(tag.text) === 0) {
      validInput = false
      notify({
        title: "Invalid CIDR",
        text: tag.text + " is not a valid IP address",
        type: 'error',
      })
    }
  })
  if(validInput) {
    formData.value.Addresses = tags.map(tag => tag.text)
  }
}

function handleChangeDns(tags) {
  let validInput = true
  tags.forEach(tag => {
    if(!isIP(tag.text)) {
      validInput = false
      notify({
        title: "Invalid IP",
        text: tag.text + " is not a valid IP address",
        type: 'error',
      })
    }
  })
  if(validInput) {
    formData.value.Dns = tags.map(tag => tag.text)
  }
}

function handleChangeDnsSearch(tags) {
  formData.value.DnsSearch = tags.map(tag => tag.text)
}

function handleChangePeerDefNetwork(tags) {
  let validInput = true
  tags.forEach(tag => {
    if(isCidr(tag.text) === 0) {
      validInput = false
      notify({
        title: "Invalid CIDR",
        text: tag.text + " is not a valid IP address",
        type: 'error',
      })
    }
  })
  if(validInput) {
    formData.value.PeerDefNetwork = tags.map(tag => tag.text)
  }
}

function handleChangePeerDefAllowedIPs(tags) {
  let validInput = true
  tags.forEach(tag => {
    if(isCidr(tag.text) === 0) {
      validInput = false
      notify({
        title: "Invalid CIDR",
        text: tag.text + " is not a valid IP address",
        type: 'error',
      })
    }
  })
  if(validInput) {
    formData.value.PeerDefAllowedIPs = tags.map(tag => tag.text)
  }
}

function handleChangePeerDefDns(tags) {
  let validInput = true
  tags.forEach(tag => {
    if(!isIP(tag.text)) {
      validInput = false
      notify({
        title: "Invalid IP",
        text: tag.text + " is not a valid IP address",
        type: 'error',
      })
    }
  })
  if(validInput) {
    formData.value.PeerDefDns = tags.map(tag => tag.text)
  }
}

function handleChangePeerDefDnsSearch(tags) {
  formData.value.PeerDefDnsSearch = tags.map(tag => tag.text)
}

async function save() {
  if (isSaving.value) return
  // Hard-block submit when AmneziaWG validation fails — easy round-trip
  // saver vs. having the kernel reject the write at MergeToPhysicalInterface.
  if (hasAwgValidationErrors.value) {
    notify({
      title: t('modals.interface-edit.amneziawg-validation-failed-title'),
      text: t('modals.interface-edit.amneziawg-validation-failed-body'),
      type: 'error',
    })
    return
  }
  isSaving.value = true
  try {
    if (props.interfaceId!=='#NEW#') {
      await interfaces.UpdateInterface(selectedInterface.value.Identifier, formData.value)
    } else {
      await interfaces.CreateInterface(formData.value)
    }
    close()
  } catch (e) {
    console.log(e)
    notify({
      title: "Failed to save interface!",
      text: e.toString(),
      type: 'error',
    })
  } finally {
    isSaving.value = false
  }
}

async function applyPeerDefaults() {
  if (props.interfaceId==='#NEW#') {
    return; // do nothing for new interfaces
  }

  if (isApplyingDefaults.value) return
  isApplyingDefaults.value = true
  try {
    await interfaces.ApplyPeerDefaults(selectedInterface.value.Identifier, formData.value)

    notify({
      title: "Peer Defaults Applied",
      text: "Applied current peer defaults to all available peers.",
      type: 'success',
    })

    await peers.LoadPeers(selectedInterface.value.Identifier) // reload all peers after applying the defaults
  } catch (e) {
    console.log(e)
    notify({
      title: "Failed to apply peer defaults!",
      text: e.toString(),
      type: 'error',
    })
  } finally {
    isApplyingDefaults.value = false
  }
}

async function del() {
  if (isDeleting.value) return
  if (!confirm(t('modals.interface-edit.confirm-delete', {id: selectedInterface.value.Identifier}))) return
  isDeleting.value = true
  try {
    await interfaces.DeleteInterface(selectedInterface.value.Identifier)

    // reload all interfaces and peers
    await interfaces.LoadInterfaces()
    if (interfaces.Count > 0 && interfaces.GetSelected !== undefined) {
      const selectedInterface = interfaces.GetSelected
      await peers.LoadPeers(selectedInterface.Identifier)
      await peers.LoadStats(selectedInterface.Identifier)
    } else {
      await peers.Reset() // reset peers if no interfaces are available
    }
    close()
  } catch (e) {
    console.log(e)
    notify({
      title: "Failed to delete interface!",
      text: e.toString(),
      type: 'error',
    })
  } finally {
    isDeleting.value = false
  }
}

</script>

<template>
  <Modal :title="title" :visible="visible" @close="close">
    <template #default>
      <ul class="nav nav-tabs">
        <li class="nav-item">
          <a class="nav-link active" data-bs-toggle="tab" href="#interface">{{ $t('modals.interface-edit.tab-interface') }}</a>
        </li>
        <li v-if="formData.Mode==='server'" class="nav-item">
          <a class="nav-link" data-bs-toggle="tab" href="#peerdefaults">{{ $t('modals.interface-edit.tab-peerdef') }}</a>
        </li>
      </ul>
      <div id="interfaceTabs" class="tab-content">
        <div id="interface" class="tab-pane fade active show">
          <fieldset>
            <legend class="mt-4">{{ $t('modals.interface-edit.header-general') }}</legend>
            <div v-if="props.interfaceId==='#NEW#'" class="form-group">
              <label class="form-label mt-4">{{ $t('modals.interface-edit.identifier.label') }}</label>
              <input v-model="formData.Identifier" class="form-control" :placeholder="$t('modals.interface-edit.identifier.placeholder')" type="text">
            </div>
            <div class="row">
              <div class="form-group col-md-6">
                <label class="form-label mt-4">{{ $t('modals.interface-edit.mode.label') }}</label>
                <select v-model="formData.Mode" class="form-select">
                  <option value="server">{{ $t('modals.interface-edit.mode.server') }}</option>
                  <option value="client">{{ $t('modals.interface-edit.mode.client') }}</option>
                  <option value="any">{{ $t('modals.interface-edit.mode.any') }}</option>
                </select>
              </div>
              <div class="form-group col-md-6">
                <label class="form-label mt-4" for="ifaceBackendSelector">{{ $t('modals.interface-edit.backend.label') }}</label>
                <select id="ifaceBackendSelector" v-model="formData.Backend" class="form-select" aria-describedby="backendHelp">
                  <option v-for="backend in settings.Setting('AvailableBackends')" :value="backend.Id">{{ backend.Id === 'local' ? $t(backend.Name) : backend.Name }}</option>
                </select>
                <small v-if="!isBackendValid" id="backendHelp" class="form-text text-warning">{{ $t('modals.interface-edit.backend.invalid-label') }}</small>
              </div>
            </div>
            <div class="form-group">
              <label class="form-label mt-4">{{ $t('modals.interface-edit.display-name.label') }}</label>
              <input v-model="formData.DisplayName" class="form-control" :placeholder="$t('modals.interface-edit.display-name.placeholder')" type="text">
            </div>
          </fieldset>
          <fieldset>
            <legend class="mt-4">{{ $t('modals.interface-edit.header-crypto') }}</legend>
            <div class="form-group">
              <label class="form-label mt-4">{{ $t('modals.interface-edit.private-key.label') }}</label>
              <input v-model="formData.PrivateKey" class="form-control" :placeholder="$t('modals.interface-edit.private-key.placeholder')" required type="text">
            </div>
            <div class="form-group">
              <label class="form-label mt-4">{{ $t('modals.interface-edit.public-key.label') }}</label>
              <input v-model="formData.PublicKey" class="form-control" :placeholder="$t('modals.interface-edit.public-key.placeholder')" required type="text">
            </div>
          </fieldset>
          <fieldset>
            <legend class="mt-4">{{ $t('modals.interface-edit.header-network') }}</legend>
            <div class="form-group">
              <label class="form-label mt-4">{{ $t('modals.interface-edit.ip.label') }}</label>
              <vue-tags-input class="form-control" v-model="currentTags.Addresses"
                              :tags="formData.Addresses.map(str => ({ text: str }))"
                              :placeholder="$t('modals.interface-edit.ip.placeholder')"
                              :validation="validateCIDR()"
                              :add-on-key="[13, 188, 32, 9]"
                              :save-on-key="[13, 188, 32, 9]"
                              :allow-edit-tags="true"
                              :separators="[',', ';', ' ']"
                              @tags-changed="handleChangeAddresses"/>
            </div>
            <div v-if="formData.Mode==='server'" class="form-group">
              <label class="form-label mt-4">{{ $t('modals.interface-edit.listen-port.label') }}</label>
              <input v-model="formData.ListenPort" class="form-control" :placeholder="$t('modals.interface-edit.listen-port.placeholder')" type="number">
            </div>
            <div v-if="formData.Mode!=='server'" class="form-group">
              <label class="form-label mt-4">{{ $t('modals.interface-edit.dns.label') }}</label>
              <vue-tags-input class="form-control" v-model="currentTags.Dns"
                              :tags="formData.Dns.map(str => ({ text: str }))"
                              :placeholder="$t('modals.interface-edit.dns.placeholder')"
                              :validation="validateIP()"
                              :add-on-key="[13, 188, 32, 9]"
                              :save-on-key="[13, 188, 32, 9]"
                              :allow-edit-tags="true"
                              :separators="[',', ';', ' ']"
                              @tags-changed="handleChangeDns"/>
            </div>
            <div v-if="formData.Mode!=='server'" class="form-group">
              <label class="form-label mt-4">{{ $t('modals.interface-edit.dns-search.label') }}</label>
              <vue-tags-input class="form-control" v-model="currentTags.DnsSearch"
                              :tags="formData.DnsSearch.map(str => ({ text: str }))"
                              :placeholder="$t('modals.interface-edit.dns-search.placeholder')"
                              :validation="validateDomain()"
                              :add-on-key="[13, 188, 32, 9]"
                              :save-on-key="[13, 188, 32, 9]"
                              :allow-edit-tags="true"
                              :separators="[',', ';', ' ']"
                              @tags-changed="handleChangeDnsSearch"/>
            </div>
            <div class="row">
              <div class="form-group col-md-6">
                <label class="form-label mt-4">{{ $t('modals.interface-edit.mtu.label') }}</label>
                <input v-model="formData.Mtu" class="form-control" :placeholder="$t('modals.interface-edit.mtu.placeholder')" type="number">
              </div>
              <div class="form-group col-md-6" v-if="formData.Backend==='local'">
                <label class="form-label mt-4">{{ $t('modals.interface-edit.firewall-mark.label') }}</label>
                <input v-model="formData.FirewallMark" class="form-control" :placeholder="$t('modals.interface-edit.firewall-mark.placeholder')" type="number">
              </div>
              <div class="form-group col-md-6" v-if="formData.Backend!=='local'">
                <label class="form-label mt-4">{{ $t('modals.interface-edit.routing-table.label') }}</label>
                <input v-model="formData.RoutingTable" aria-describedby="routingTableHelp" class="form-control" :placeholder="$t('modals.interface-edit.routing-table.placeholder')" type="text">
                <small id="routingTableHelp" class="form-text text-muted">{{ $t('modals.interface-edit.routing-table.description') }}</small>
              </div>
              <div class="form-group col-md-6" v-else>
              </div>
            </div>
            <div class="row" v-if="formData.Backend==='local'">
              <div class="form-group col-md-6">
                <label class="form-label mt-4">{{ $t('modals.interface-edit.routing-table.label') }}</label>
                <input v-model="formData.RoutingTable" aria-describedby="routingTableHelp" class="form-control" :placeholder="$t('modals.interface-edit.routing-table.placeholder')" type="text">
                <small id="routingTableHelp" class="form-text text-muted">{{ $t('modals.interface-edit.routing-table.description') }}</small>
              </div>
              <div class="form-group col-md-6">
              </div>
            </div>
          </fieldset>

          <!-- AmneziaWG V2 obfuscation parameters. Hidden unless backend
               selected as 'amneziawg'. Each input has an inline tooltip
               (title=) summarizing the param + recommended range, and a
               red error span below when client-side validation fails. -->
          <!-- User-pool config (BNet-m76e). Per-(user × interface)
               auto-allocator settings. Empty supernet on a family
               disables auto-allocation for that family. -->
          <fieldset>
            <legend class="mt-4">User-pool auto-allocator</legend>

            <!-- Allocator state panel — read-only. Only shown for
                 existing interfaces (props.interfaceId !== '#NEW#'),
                 since pool state has no meaning before the iface row
                 exists. (BNet-m76e QoL #3) -->
            <div v-if="props.interfaceId !== '#NEW#' && interfaces.PoolState"
                 class="alert alert-info py-2 mb-3">
              <div class="row small">
                <div class="col-md-4">
                  <strong>Users with pools:</strong>
                  {{ interfaces.PoolState.AllocatedCount }}
                </div>
                <div class="col-md-4">
                  <strong>Next free /v4:</strong>
                  <code v-if="interfaces.PoolState.NextFreeV4">{{ interfaces.PoolState.NextFreeV4 }}</code>
                  <em v-else>none / supernet exhausted</em>
                </div>
                <div class="col-md-4">
                  <strong>Slices total / reserved:</strong>
                  v4 {{ interfaces.PoolState.SupernetV4Total }}/{{ interfaces.PoolState.SupernetV4Reserved }}
                </div>
              </div>
              <div class="row small mt-1">
                <div class="col-md-6">
                  <strong>Next free /ULA:</strong>
                  <code v-if="interfaces.PoolState.NextFreeV6Ula">{{ interfaces.PoolState.NextFreeV6Ula }}</code>
                  <em v-else>—</em>
                </div>
                <div class="col-md-6">
                  <strong>Next free /PI:</strong>
                  <code v-if="interfaces.PoolState.NextFreeV6Pi">{{ interfaces.PoolState.NextFreeV6Pi }}</code>
                  <em v-else>—</em>
                </div>
              </div>
            </div>

            <p class="text-muted small">
              Per-(user × interface) /N slices auto-allocated on first
              peer creation. Each user's peers draw from their own slice
              within the supernet, skipping reserved CIDRs. Empty
              supernet disables auto-allocation for that family.
            </p>
            <div class="row">
              <div class="form-group col-md-6">
                <label class="form-label mt-2">Supernet IPv4</label>
                <input v-model="formData.UserPool.SupernetV4"
                       class="form-control"
                       :class="{'is-invalid': cidrError(formData.UserPool.SupernetV4, 'v4'),
                                'is-valid': formData.UserPool.SupernetV4 && !cidrError(formData.UserPool.SupernetV4, 'v4')}"
                       placeholder="10.66.0.0/16">
                <div class="invalid-feedback">{{ cidrError(formData.UserPool.SupernetV4, 'v4') }}</div>
              </div>
              <div class="form-group col-md-2">
                <label class="form-label mt-2">Slice /N</label>
                <input v-model.number="formData.UserPool.SizeV4" type="number" class="form-control" placeholder="27">
              </div>
              <div class="form-group col-md-4">
                <label class="form-label mt-2">Reserved (comma)</label>
                <input :value="(formData.UserPool.ReservedV4 || []).join(',')"
                       @input="formData.UserPool.ReservedV4 = $event.target.value.split(',').map(s => s.trim()).filter(Boolean)"
                       class="form-control" placeholder="10.66.255.0/24,10.66.254.0/24">
              </div>
            </div>
            <div class="row">
              <div class="form-group col-md-6">
                <label class="form-label mt-2">Supernet IPv6 ULA</label>
                <input v-model="formData.UserPool.SupernetV6Ula"
                       class="form-control"
                       :class="{'is-invalid': cidrError(formData.UserPool.SupernetV6Ula, 'v6'),
                                'is-valid': formData.UserPool.SupernetV6Ula && !cidrError(formData.UserPool.SupernetV6Ula, 'v6')}"
                       placeholder="fdcc:ad94:bacf:6160::/64">
                <div class="invalid-feedback">{{ cidrError(formData.UserPool.SupernetV6Ula, 'v6') }}</div>
              </div>
              <div class="form-group col-md-2">
                <label class="form-label mt-2">Slice /N</label>
                <input v-model.number="formData.UserPool.SizeV6Ula" type="number" class="form-control" placeholder="80">
              </div>
              <div class="form-group col-md-4">
                <label class="form-label mt-2">Reserved (comma)</label>
                <input :value="(formData.UserPool.ReservedV6Ula || []).join(',')"
                       @input="formData.UserPool.ReservedV6Ula = $event.target.value.split(',').map(s => s.trim()).filter(Boolean)"
                       class="form-control" placeholder="fdcc:...:ffff::/96">
              </div>
            </div>
            <div class="row">
              <div class="form-group col-md-6">
                <label class="form-label mt-2">Supernet IPv6 PI</label>
                <input v-model="formData.UserPool.SupernetV6Pi"
                       class="form-control"
                       :class="{'is-invalid': cidrError(formData.UserPool.SupernetV6Pi, 'v6'),
                                'is-valid': formData.UserPool.SupernetV6Pi && !cidrError(formData.UserPool.SupernetV6Pi, 'v6')}"
                       placeholder="2602:f481:0:cc::/64">
                <div class="invalid-feedback">{{ cidrError(formData.UserPool.SupernetV6Pi, 'v6') }}</div>
              </div>
              <div class="form-group col-md-2">
                <label class="form-label mt-2">Slice /N</label>
                <input v-model.number="formData.UserPool.SizeV6Pi" type="number" class="form-control" placeholder="80">
              </div>
              <div class="form-group col-md-4">
                <label class="form-label mt-2">Reserved (comma)</label>
                <input :value="(formData.UserPool.ReservedV6Pi || []).join(',')"
                       @input="formData.UserPool.ReservedV6Pi = $event.target.value.split(',').map(s => s.trim()).filter(Boolean)"
                       class="form-control" placeholder="2602:...:ffff::/96">
              </div>
            </div>
          </fieldset>

          <fieldset v-if="formData.Backend==='amneziawg'">
            <legend class="mt-4">{{ $t('modals.interface-edit.header-amneziawg') }}</legend>
            <p class="text-muted small">{{ $t('modals.interface-edit.amneziawg-description') }}</p>

            <!-- Generate-random button: sane defaults for all numeric params + 4 distinct H values.
                 Confirmation via subsequent notify so operators don't accidentally clobber tuned values. -->
            <div class="d-flex justify-content-end mb-2">
              <button type="button" class="btn btn-outline-secondary btn-sm" @click.prevent="generateRandomAwgParams">
                <i class="fas fa-dice me-1"></i>{{ $t('modals.interface-edit.amneziawg-randomize-button') }}
              </button>
            </div>

            <!-- Junk packets row -->
            <div class="row">
              <div class="form-group col-md-4">
                <label class="form-label mt-3">Jc <i class="fas fa-question-circle text-muted small" :title="awgParamSpec.Jc.description"></i></label>
                <input v-model.number="formData.AmneziaWG.Jc" type="number" :min="awgParamSpec.Jc.min" :max="awgParamSpec.Jc.max"
                       :class="['form-control', awgValidationErrors.Jc ? 'is-invalid' : '']"
                       :placeholder="`${awgParamSpec.Jc.recommended[0]}–${awgParamSpec.Jc.recommended[1]}`">
                <div v-if="awgValidationErrors.Jc" class="invalid-feedback">{{ awgValidationErrors.Jc }}</div>
              </div>
              <div class="form-group col-md-4">
                <label class="form-label mt-3">Jmin <i class="fas fa-question-circle text-muted small" :title="awgParamSpec.Jmin.description"></i></label>
                <input v-model.number="formData.AmneziaWG.Jmin" type="number" :min="awgParamSpec.Jmin.min" :max="awgParamSpec.Jmin.max"
                       :class="['form-control', awgValidationErrors.Jmin ? 'is-invalid' : '']"
                       :placeholder="`${awgParamSpec.Jmin.recommended[0]}–${awgParamSpec.Jmin.recommended[1]}`">
                <div v-if="awgValidationErrors.Jmin" class="invalid-feedback">{{ awgValidationErrors.Jmin }}</div>
              </div>
              <div class="form-group col-md-4">
                <label class="form-label mt-3">Jmax <i class="fas fa-question-circle text-muted small" :title="awgParamSpec.Jmax.description"></i></label>
                <input v-model.number="formData.AmneziaWG.Jmax" type="number" :min="awgParamSpec.Jmax.min" :max="awgParamSpec.Jmax.max"
                       :class="['form-control', awgValidationErrors.Jmax ? 'is-invalid' : '']"
                       :placeholder="`${awgParamSpec.Jmax.recommended[0]}–${awgParamSpec.Jmax.recommended[1]}`">
                <div v-if="awgValidationErrors.Jmax" class="invalid-feedback">{{ awgValidationErrors.Jmax }}</div>
              </div>
            </div>

            <!-- Packet stuffing row (S1-S4) -->
            <div class="row">
              <div class="form-group col-md-3" v-for="key in ['S1','S2','S3','S4']" :key="key">
                <label class="form-label mt-3">{{ key }} <i class="fas fa-question-circle text-muted small" :title="awgParamSpec[key].description"></i></label>
                <input v-model.number="formData.AmneziaWG[key]" type="number" :min="awgParamSpec[key].min" :max="awgParamSpec[key].max"
                       :class="['form-control', awgValidationErrors[key] ? 'is-invalid' : '']"
                       :placeholder="`${awgParamSpec[key].recommended[0]}–${awgParamSpec[key].recommended[1]}`">
                <div v-if="awgValidationErrors[key]" class="invalid-feedback">{{ awgValidationErrors[key] }}</div>
              </div>
            </div>

            <!-- Header magic row (H1-H4) -->
            <div class="row">
              <div class="form-group col-md-3" v-for="key in ['H1','H2','H3','H4']" :key="key">
                <label class="form-label mt-3">{{ key }} <i class="fas fa-question-circle text-muted small" :title="awgParamSpec[key].description"></i></label>
                <input v-model.number="formData.AmneziaWG[key]" type="number" :min="awgParamSpec[key].min" :max="awgParamSpec[key].max"
                       :class="['form-control', awgValidationErrors[key] ? 'is-invalid' : '']"
                       placeholder="random uint32 > 4">
                <div v-if="awgValidationErrors[key]" class="invalid-feedback">{{ awgValidationErrors[key] }}</div>
              </div>
            </div>

            <!-- Injection bytes row (I1-I5) — V2 only, hex strings -->
            <div class="row">
              <div class="form-group col-md-12 mt-3">
                <p class="text-muted small mb-1">{{ $t('modals.interface-edit.amneziawg-i-description') }}</p>
              </div>
              <div class="form-group col-md-12" v-for="key in ['I1','I2','I3','I4','I5']" :key="key">
                <label class="form-label mt-2">{{ key }} <i class="fas fa-question-circle text-muted small" :title="awgParamSpec[key].description"></i></label>
                <input v-model="formData.AmneziaWG[key]" type="text" maxlength="200"
                       :class="['form-control font-monospace', awgValidationErrors[key] ? 'is-invalid' : '']"
                       placeholder="hex string (e.g. deadbeef) — empty to disable">
                <div v-if="awgValidationErrors[key]" class="invalid-feedback">{{ awgValidationErrors[key] }}</div>
              </div>
            </div>
          </fieldset>

          <fieldset v-if="formData.Backend==='local'">
            <legend class="mt-4">{{ $t('modals.interface-edit.header-hooks') }}</legend>
            <div class="form-group">
              <label class="form-label mt-4">{{ $t('modals.interface-edit.pre-up.label') }}</label>
              <textarea v-model="formData.PreUp" class="form-control" rows="2" :placeholder="$t('modals.interface-edit.pre-up.placeholder')"></textarea>
            </div>
            <div class="form-group">
              <label class="form-label mt-4">{{ $t('modals.interface-edit.post-up.label') }}</label>
              <textarea v-model="formData.PostUp" class="form-control" rows="2" :placeholder="$t('modals.interface-edit.post-up.placeholder')"></textarea>
            </div>
            <div class="form-group">
              <label class="form-label mt-4">{{ $t('modals.interface-edit.pre-down.label') }}</label>
              <textarea v-model="formData.PreDown" class="form-control" rows="2" :placeholder="$t('modals.interface-edit.pre-down.placeholder')"></textarea>
            </div>
            <div class="form-group">
              <label class="form-label mt-4">{{ $t('modals.interface-edit.post-down.label') }}</label>
              <textarea v-model="formData.PostDown" class="form-control" rows="2" :placeholder="$t('modals.interface-edit.post-down.placeholder')"></textarea>
            </div>
          </fieldset>
          <fieldset>
            <legend class="mt-4">{{ $t('modals.interface-edit.header-state') }}</legend>
            <div class="form-check form-switch">
              <input v-model="formData.Disabled" class="form-check-input" type="checkbox">
              <label class="form-check-label">{{ $t('modals.interface-edit.disabled.label') }}</label>
            </div>
            <div class="form-check form-switch" v-if="formData.Mode==='server' && settings.Setting('CreateDefaultPeer')">
              <input v-model="formData.CreateDefaultPeer" class="form-check-input" type="checkbox">
              <label class="form-check-label">{{ $t('modals.interface-edit.create-default-peer.label') }}</label>
            </div>
            <div class="form-check form-switch" v-if="formData.Backend==='local'">
              <input v-model="formData.SaveConfig" checked="" class="form-check-input" type="checkbox">
              <label class="form-check-label">{{ $t('modals.interface-edit.save-config.label') }}</label>
            </div>
          </fieldset>
        </div>
        <div id="peerdefaults" class="tab-pane fade">
          <fieldset>
            <legend class="mt-4">{{ $t('modals.interface-edit.header-network') }}</legend>
            <div class="form-group">
              <label class="form-label mt-4">{{ $t('modals.interface-edit.defaults.endpoint.label') }}</label>
              <input v-model="formData.PeerDefEndpoint" class="form-control" :placeholder="$t('modals.interface-edit.defaults.endpoint.placeholder')" type="text">
              <small class="form-text text-muted">{{ $t('modals.interface-edit.defaults.endpoint.description') }}</small>
            </div>
            <div class="form-group">
              <label class="form-label mt-4">{{ $t('modals.interface-edit.defaults.networks.label') }}</label>
              <vue-tags-input class="form-control" v-model="currentTags.PeerDefNetwork"
                              :tags="formData.PeerDefNetwork.map(str => ({ text: str }))"
                              :placeholder="$t('modals.interface-edit.defaults.networks.placeholder')"
                              :validation="validateCIDR()"
                              :add-on-key="[13, 188, 32, 9]"
                              :save-on-key="[13, 188, 32, 9]"
                              :allow-edit-tags="true"
                              :separators="[',', ';', ' ']"
                              @tags-changed="handleChangePeerDefNetwork"/>
              <small class="form-text text-muted">{{ $t('modals.interface-edit.defaults.networks.description') }}</small>
            </div>
            <div class="form-group">
              <label class="form-label mt-4">{{ $t('modals.interface-edit.defaults.allowed-ip.label') }}</label>
              <vue-tags-input class="form-control" v-model="currentTags.PeerDefAllowedIPs"
                              :tags="formData.PeerDefAllowedIPs.map(str => ({ text: str }))"
                              :placeholder="$t('modals.interface-edit.defaults.allowed-ip.placeholder')"
                              :validation="validateCIDR()"
                              :add-on-key="[13, 188, 32, 9]"
                              :save-on-key="[13, 188, 32, 9]"
                              :allow-edit-tags="true"
                              :separators="[',', ';', ' ']"
                              @tags-changed="handleChangePeerDefAllowedIPs"/>
            </div>
            <div class="form-group">
              <label class="form-label mt-4">{{ $t('modals.interface-edit.dns.label') }}</label>
              <vue-tags-input class="form-control" v-model="currentTags.PeerDefDns"
                              :tags="formData.PeerDefDns.map(str => ({ text: str }))"
                              :placeholder="$t('modals.interface-edit.dns.placeholder')"
                              :validation="validateIP()"
                              :add-on-key="[13, 188, 32, 9]"
                              :save-on-key="[13, 188, 32, 9]"
                              :allow-edit-tags="true"
                              :separators="[',', ';', ' ']"
                              @tags-changed="handleChangePeerDefDns"/>
            </div>
            <div class="form-group">
              <label class="form-label mt-4">{{ $t('modals.interface-edit.dns-search.label') }}</label>
              <vue-tags-input class="form-control" v-model="currentTags.PeerDefDnsSearch"
                              :tags="formData.PeerDefDnsSearch.map(str => ({ text: str }))"
                              :placeholder="$t('modals.interface-edit.dns-search.placeholder')"
                              :validation="validateDomain()"
                              :add-on-key="[13, 188, 32, 9]"
                              :save-on-key="[13, 188, 32, 9]"
                              :allow-edit-tags="true"
                              :separators="[',', ';', ' ']"
                              @tags-changed="handleChangePeerDefDnsSearch"/>
            </div>
            <div class="row">
              <div class="form-group col-md-6">
                <label class="form-label mt-4">{{ $t('modals.interface-edit.defaults.mtu.label') }}</label>
                <input v-model="formData.PeerDefMtu" class="form-control" :placeholder="$t('modals.interface-edit.defaults.mtu.placeholder')" type="number">
              </div>
              <div class="form-group col-md-6">
                <label class="form-label mt-4">{{ $t('modals.interface-edit.firewall-mark.label') }}</label>
                <input v-model="formData.PeerDefFirewallMark" class="form-control" :placeholder="$t('modals.interface-edit.firewall-mark.placeholder')" type="number">
              </div>
            </div>
            <div class="row">
              <div class="form-group col-md-6">
                <label class="form-label mt-4">{{ $t('modals.interface-edit.routing-table.label') }}</label>
                <input v-model="formData.PeerDefRoutingTable" class="form-control" :placeholder="$t('modals.interface-edit.routing-table.placeholder')" type="number">
              </div>
              <div class="form-group col-md-6">
                <label class="form-label mt-4">{{ $t('modals.interface-edit.defaults.keep-alive.label') }}</label>
                <input v-model="formData.PeerDefPersistentKeepalive" class="form-control" :placeholder="$t('modals.interface-edit.defaults.keep-alive.placeholder')" type="number">
              </div>
            </div>
          </fieldset>
          <fieldset>
            <legend class="mt-4">{{ $t('modals.interface-edit.header-peer-hooks') }}</legend>
            <div class="form-group">
              <label class="form-label mt-4">{{ $t('modals.interface-edit.pre-up.label') }}</label>
              <textarea v-model="formData.PeerDefPreUp" class="form-control" rows="2" :placeholder="$t('modals.interface-edit.pre-up.placeholder')"></textarea>
            </div>
            <div class="form-group">
              <label class="form-label mt-4">{{ $t('modals.interface-edit.post-up.label') }}</label>
              <textarea v-model="formData.PeerDefPostUp" class="form-control" rows="2" :placeholder="$t('modals.interface-edit.post-up.placeholder')"></textarea>
            </div>
            <div class="form-group">
              <label class="form-label mt-4">{{ $t('modals.interface-edit.pre-down.label') }}</label>
              <textarea v-model="formData.PeerDefPreDown" class="form-control" rows="2" :placeholder="$t('modals.interface-edit.pre-down.placeholder')"></textarea>
            </div>
            <div class="form-group">
              <label class="form-label mt-4">{{ $t('modals.interface-edit.post-down.label') }}</label>
              <textarea v-model="formData.PeerDefPostDown" class="form-control" rows="2" :placeholder="$t('modals.interface-edit.post-down.placeholder')"></textarea>
            </div>
          </fieldset>
          <fieldset v-if="props.interfaceId!=='#NEW#'" class="text-end">
            <hr class="mt-4">
            <button class="btn btn-primary me-1" type="button" @click.prevent="applyPeerDefaults" :disabled="isApplyingDefaults">
              <span v-if="isApplyingDefaults" class="spinner-border spinner-border-sm me-1" role="status" aria-hidden="true"></span>
              {{ $t('modals.interface-edit.button-apply-defaults') }}
            </button>
          </fieldset>
        </div>
      </div>
    </template>
    <template #footer>
      <div class="flex-fill text-start">
        <button v-if="props.interfaceId!=='#NEW#'" class="btn btn-danger me-1" type="button" @click.prevent="del" :disabled="isDeleting">
          <span v-if="isDeleting" class="spinner-border spinner-border-sm me-1" role="status" aria-hidden="true"></span>
          {{ $t('general.delete') }}
        </button>
      </div>
      <button class="btn btn-primary me-1" type="button" @click.prevent="save" :disabled="isSaving">
        <span v-if="isSaving" class="spinner-border spinner-border-sm me-1" role="status" aria-hidden="true"></span>
        {{ $t('general.save') }}
      </button>
      <button class="btn btn-secondary" type="button" @click.prevent="close">{{ $t('general.close') }}</button>
    </template>
  </Modal>
</template>

<style>

</style>
