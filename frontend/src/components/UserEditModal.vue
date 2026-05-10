<script setup>
import Modal from "./Modal.vue";
import {userStore} from "@/stores/users";
import {computed, ref, watch} from "vue";
import { useI18n } from 'vue-i18n';
import { notify } from "@kyvg/vue3-notification";
import {freshUser} from "@/helpers/models";
import {settingsStore} from "@/stores/settings";
import {interfaceStore} from "@/stores/interfaces";
import {cidrError, cidrContains} from "@/helpers/cidr";

const { t } = useI18n()

const users = userStore()
const settings = settingsStore()
const interfaces = interfaceStore()

const props = defineProps({
  userId: String,
  visible: Boolean,
})

const emit = defineEmits(['close'])

const selectedUser = computed(() => {
  return users.Find(props.userId)
})

const title = computed(() => {
  if (!props.visible) {
    return "" // otherwise interfaces.GetSelected will die...
  }
  if (selectedUser.value) {
    return t("modals.user-edit.headline-edit") + " " + selectedUser.value.Identifier
  }
  return t("modals.user-edit.headline-new")
})

const formData = ref(freshUser())
const isSaving = ref(false)
const isDeleting = ref(false)

const passwordWeak = computed(() => {
  return formData.value.Password && formData.value.Password.length > 0 && formData.value.Password.length < settings.Setting('MinPasswordLength')
})

const formValid = computed(() => {
  if (!formData.value.AuthSources.some(s => s === 'db')) {
    return true // nothing to validate
  }
  if (props.userId !== '#NEW#' && passwordWeak.value) {
    return false
  }
  if (props.userId === '#NEW#' && (!formData.value.Password || formData.value.Password.length < 1)) {
    return false
  }
  if (props.userId === '#NEW#' && passwordWeak.value) {
    return false
  }
  if (!formData.value.Identifier || formData.value.Identifier.length < 1) {
    return false
  }
  return true
})


// functions

watch(() => props.visible, async (newValue, oldValue) => {
      if (oldValue === false && newValue === true) { // if modal is shown
        if (!selectedUser.value) {
          formData.value = freshUser()
        } else { // fill existing userdata
          formData.value.Identifier = selectedUser.value.Identifier
          formData.value.Email = selectedUser.value.Email
          formData.value.AuthSources = selectedUser.value.AuthSources
          formData.value.IsAdmin = selectedUser.value.IsAdmin
          formData.value.Firstname = selectedUser.value.Firstname
          formData.value.Lastname = selectedUser.value.Lastname
          formData.value.Phone = selectedUser.value.Phone
          formData.value.Department = selectedUser.value.Department
          formData.value.Notes = selectedUser.value.Notes
          formData.value.Password = ""
          formData.value.Disabled = selectedUser.value.Disabled
          formData.value.Locked = selectedUser.value.Locked
          formData.value.PersistLocalChanges = selectedUser.value.PersistLocalChanges
          // Load per-(user × interface) pools (BNet-m76e).
          if (selectedUser.value.Identifier) {
            await users.LoadUserPools(selectedUser.value.Identifier)
          }
        }
      }
    }
)

// Pool editing state — one entry per pool row, keyed by interface_id.
// Holds in-flight (unsaved) changes; reverts to store on cancel.
const poolDrafts = ref({})
const savingPool = ref({})

function poolDraft(ifaceId) {
  if (!poolDrafts.value[ifaceId]) {
    const existing = users.Pools.find(p => p.InterfaceIdentifier === ifaceId)
    poolDrafts.value[ifaceId] = {
      PoolV4: existing?.PoolV4 || "",
      PoolV6Ula: existing?.PoolV6Ula || "",
      PoolV6Pi: existing?.PoolV6Pi || "",
    }
  }
  return poolDrafts.value[ifaceId]
}

// Live validation for a pool field. Checks CIDR shape AND containment
// inside the iface's supernet (when known). Returns "" when valid or
// empty (server side has the final say on overlap with reserved/other-
// user pools).
function poolFieldError(ifaceId, family, value) {
  if (!value) return ""
  const flavor = family === 'v4' ? 'v4' : 'v6'
  const shapeErr = cidrError(value, flavor)
  if (shapeErr) return shapeErr
  // Look up iface supernet for containment check.
  const iface = interfaces.Find(ifaceId)
  if (!iface || !iface.UserPool) return ""
  const supernet =
    family === 'v4' ? iface.UserPool.SupernetV4 :
    family === 'ula' ? iface.UserPool.SupernetV6Ula :
    family === 'pi' ? iface.UserPool.SupernetV6Pi : ""
  if (supernet && !cidrContains(supernet, value)) {
    return `not within ${supernet}`
  }
  return ""
}

function poolDraftHasError(ifaceId) {
  const d = poolDrafts.value[ifaceId]
  if (!d) return false
  return !!(poolFieldError(ifaceId, 'v4', d.PoolV4) ||
            poolFieldError(ifaceId, 'ula', d.PoolV6Ula) ||
            poolFieldError(ifaceId, 'pi', d.PoolV6Pi))
}

async function savePool(ifaceId) {
  if (!selectedUser.value) return
  savingPool.value[ifaceId] = true
  try {
    await users.UpdateUserPool(selectedUser.value.Identifier, ifaceId, poolDrafts.value[ifaceId])
    delete poolDrafts.value[ifaceId]
  } finally {
    savingPool.value[ifaceId] = false
  }
}

async function releasePool(ifaceId) {
  if (!selectedUser.value) return
  if (!confirm(`Release pool for ${selectedUser.value.Identifier} on ${ifaceId}? Existing peer addresses are NOT renumbered — auto-allocator will pick a fresh slot on next peer creation.`)) return
  savingPool.value[ifaceId] = true
  try {
    await users.ReleaseUserPool(selectedUser.value.Identifier, ifaceId)
    delete poolDrafts.value[ifaceId]
  } finally {
    savingPool.value[ifaceId] = false
  }
}

function close() {
  formData.value = freshUser()
  emit('close')
}

async function save() {
  if (isSaving.value) return
  isSaving.value = true
  try {
    if (props.userId!=='#NEW#') {
      await users.UpdateUser(selectedUser.value.Identifier, formData.value)
    } else {
      await users.CreateUser(formData.value)
    }
    close()
  } catch (e) {
    notify({
      title: "Failed to save user!",
      text: e.toString(),
      type: 'error',
    })
  } finally {
    isSaving.value = false
  }
}

async function del() {
  if (isDeleting.value) return
  if (!confirm(t('modals.user-edit.confirm-delete', {id: selectedUser.value.Identifier}))) return
  isDeleting.value = true
  try {
    await users.DeleteUser(selectedUser.value.Identifier)
    close()
  } catch (e) {
    notify({
      title: "Failed to delete user!",
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
      <fieldset>
        <legend class="mt-4">{{ $t('modals.user-edit.header-general') }}</legend>
        <div v-if="props.userId==='#NEW#'" class="form-group">
          <label class="form-label mt-4">{{ $t('modals.user-edit.identifier.label') }}</label>
          <input v-model="formData.Identifier" class="form-control" :placeholder="$t('modals.user-edit.identifier.placeholder')" type="text">
        </div>
        <div class="form-group">
          <label class="form-label mt-4">{{ $t('modals.user-edit.source.label') }}</label>
          <input v-model="formData.AuthSources" class="form-control" disabled="disabled" :placeholder="$t('modals.user-edit.source.placeholder')" type="text">
        </div>
        <div class="form-group" v-if="formData.AuthSources.some(s => s ==='db')">
          <label class="form-label mt-4">{{ $t('modals.user-edit.password.label') }}</label>
          <input v-model="formData.Password" aria-describedby="passwordHelp" class="form-control" :class="{ 'is-invalid': passwordWeak,  'is-valid': formData.Password !== '' && !passwordWeak }" :placeholder="$t('modals.user-edit.password.placeholder')" type="password">
          <div class="invalid-feedback">{{ $t('modals.user-edit.password.too-weak') }}</div>
          <small v-if="props.userId!=='#NEW#'" id="passwordHelp" class="form-text text-muted">{{ $t('modals.user-edit.password.description') }}</small>
        </div>
      </fieldset>
      <fieldset v-if="formData.AuthSources.some(s => s !=='db') && !formData.PersistLocalChanges">
        <legend class="mt-4">{{ $t('modals.user-edit.header-personal') }}</legend>
        <div class="alert alert-warning mt-3">
          {{ $t('modals.user-edit.sync-warning') }}
        </div>
      </fieldset>
      <fieldset v-if="!formData.AuthSources.some(s => s !=='db') || formData.PersistLocalChanges">
        <legend class="mt-4">{{ $t('modals.user-edit.header-personal') }}</legend>
        <div class="form-group">
          <label class="form-label mt-4">{{ $t('modals.user-edit.email.label') }}</label>
          <input v-model="formData.Email" class="form-control" :placeholder="$t('modals.user-edit.email.placeholder')" type="email">
        </div>
        <div class="row">
          <div class="form-group col-md-6">
            <label class="form-label mt-4">{{ $t('modals.user-edit.firstname.label') }}</label>
            <input v-model="formData.Firstname" class="form-control" :placeholder="$t('modals.user-edit.firstname.placeholder')" type="text">
          </div>
          <div class="form-group col-md-6">
            <label class="form-label mt-4">{{ $t('modals.user-edit.lastname.label') }}</label>
            <input v-model="formData.Lastname" class="form-control" :placeholder="$t('modals.user-edit.lastname.placeholder')" type="text">
          </div>
        </div>
        <div class="row">
          <div class="form-group col-md-6">
            <label class="form-label mt-4">{{ $t('modals.user-edit.phone.label') }}</label>
            <input v-model="formData.Phone" class="form-control" :placeholder="$t('modals.user-edit.phone.placeholder')" type="text">
          </div>
          <div class="form-group col-md-6">
            <label class="form-label mt-4">{{ $t('modals.user-edit.department.label') }}</label>
            <input v-model="formData.Department" class="form-control" :placeholder="$t('modals.user-edit.department.placeholder')" type="text">
          </div>
        </div>
      </fieldset>
      <fieldset>
        <legend class="mt-4">{{ $t('modals.user-edit.header-notes') }}</legend>
        <div class="form-group">
          <label class="form-label mt-4">{{ $t('modals.user-edit.notes.label') }}</label>
          <textarea v-model="formData.Notes" class="form-control" rows="2"></textarea>
        </div>
      </fieldset>
      <fieldset v-if="props.userId !== '#NEW#'">
        <legend class="mt-4">Network pools</legend>
        <p class="text-muted small mb-2">
          Per-interface /N slices for this user. Empty = not allocated;
          first peer creation auto-allocates from the interface's supernet.
          Changing a pool auto-renumbers existing peer addresses (host
          bits preserved).
        </p>
        <div v-if="users.Pools.length === 0" class="text-muted">
          <em>No pools allocated yet.</em>
        </div>
        <div v-for="pool in users.Pools" :key="pool.InterfaceIdentifier"
             class="card mt-2">
          <div class="card-body">
            <h6 class="card-subtitle mb-2 text-muted">
              <code>{{ pool.InterfaceIdentifier }}</code>
            </h6>
            <div class="row g-2">
              <div class="col-md-4">
                <label class="form-label small">IPv4 pool</label>
                <input v-model="poolDraft(pool.InterfaceIdentifier).PoolV4"
                       class="form-control form-control-sm"
                       :class="{'is-invalid': poolFieldError(pool.InterfaceIdentifier, 'v4', poolDraft(pool.InterfaceIdentifier).PoolV4),
                                'is-valid': poolDraft(pool.InterfaceIdentifier).PoolV4 && !poolFieldError(pool.InterfaceIdentifier, 'v4', poolDraft(pool.InterfaceIdentifier).PoolV4)}"
                       :placeholder="pool.PoolV4 || '10.66.0.0/27'">
                <div class="invalid-feedback">{{ poolFieldError(pool.InterfaceIdentifier, 'v4', poolDraft(pool.InterfaceIdentifier).PoolV4) }}</div>
              </div>
              <div class="col-md-4">
                <label class="form-label small">IPv6 ULA pool</label>
                <input v-model="poolDraft(pool.InterfaceIdentifier).PoolV6Ula"
                       class="form-control form-control-sm"
                       :class="{'is-invalid': poolFieldError(pool.InterfaceIdentifier, 'ula', poolDraft(pool.InterfaceIdentifier).PoolV6Ula),
                                'is-valid': poolDraft(pool.InterfaceIdentifier).PoolV6Ula && !poolFieldError(pool.InterfaceIdentifier, 'ula', poolDraft(pool.InterfaceIdentifier).PoolV6Ula)}"
                       :placeholder="pool.PoolV6Ula || 'fdcc:...:0/80'">
                <div class="invalid-feedback">{{ poolFieldError(pool.InterfaceIdentifier, 'ula', poolDraft(pool.InterfaceIdentifier).PoolV6Ula) }}</div>
              </div>
              <div class="col-md-4">
                <label class="form-label small">IPv6 PI pool</label>
                <input v-model="poolDraft(pool.InterfaceIdentifier).PoolV6Pi"
                       class="form-control form-control-sm"
                       :class="{'is-invalid': poolFieldError(pool.InterfaceIdentifier, 'pi', poolDraft(pool.InterfaceIdentifier).PoolV6Pi),
                                'is-valid': poolDraft(pool.InterfaceIdentifier).PoolV6Pi && !poolFieldError(pool.InterfaceIdentifier, 'pi', poolDraft(pool.InterfaceIdentifier).PoolV6Pi)}"
                       :placeholder="pool.PoolV6Pi || '2602:...:0/80'">
                <div class="invalid-feedback">{{ poolFieldError(pool.InterfaceIdentifier, 'pi', poolDraft(pool.InterfaceIdentifier).PoolV6Pi) }}</div>
              </div>
            </div>
            <div class="mt-2">
              <button class="btn btn-sm btn-primary me-1" type="button"
                      :disabled="savingPool[pool.InterfaceIdentifier] || poolDraftHasError(pool.InterfaceIdentifier)"
                      @click.prevent="savePool(pool.InterfaceIdentifier)">
                <span v-if="savingPool[pool.InterfaceIdentifier]"
                      class="spinner-border spinner-border-sm me-1"></span>
                Save
              </button>
              <button class="btn btn-sm btn-outline-danger" type="button"
                      :disabled="savingPool[pool.InterfaceIdentifier]"
                      @click.prevent="releasePool(pool.InterfaceIdentifier)">
                Release
              </button>
              <small class="text-muted ms-2">
                Last update: {{ pool.UpdatedAt ? new Date(pool.UpdatedAt).toLocaleString() : '—' }}
                <span v-if="pool.UpdatedBy"> by {{ pool.UpdatedBy }}</span>
              </small>
            </div>
          </div>
        </div>
      </fieldset>

      <fieldset>
        <legend class="mt-4">{{ $t('modals.user-edit.header-state') }}</legend>
        <div class="form-check form-switch">
          <input v-model="formData.Disabled" class="form-check-input" type="checkbox">
          <label class="form-check-label" >{{ $t('modals.user-edit.disabled.label') }}</label>
        </div>
        <div class="form-check form-switch">
          <input v-model="formData.Locked" class="form-check-input" type="checkbox">
          <label class="form-check-label" >{{ $t('modals.user-edit.locked.label') }}</label>
        </div>
        <div class="form-check form-switch" v-if="!formData.AuthSources.some(s => s !=='db') || formData.PersistLocalChanges">
          <input v-model="formData.IsAdmin" checked="" class="form-check-input" type="checkbox">
          <label class="form-check-label">{{ $t('modals.user-edit.admin.label') }}</label>
        </div>
        <div class="form-check form-switch" v-if="formData.AuthSources.some(s => s !=='db')">
          <input v-model="formData.PersistLocalChanges" class="form-check-input" type="checkbox">
          <label class="form-check-label" >{{ $t('modals.user-edit.persist-local-changes.label') }}</label>
        </div>
      </fieldset>

    </template>
    <template #footer>
      <div class="flex-fill text-start">
        <button v-if="props.userId!=='#NEW#'" class="btn btn-danger me-1" type="button" @click.prevent="del" :disabled="isDeleting">
          <span v-if="isDeleting" class="spinner-border spinner-border-sm me-1" role="status" aria-hidden="true"></span>
          {{ $t('general.delete') }}
        </button>
      </div>
      <button class="btn btn-primary me-1" type="button" @click.prevent="save" :disabled="!formValid || isSaving">
        <span v-if="isSaving" class="spinner-border spinner-border-sm me-1" role="status" aria-hidden="true"></span>
        {{ $t('general.save') }}
      </button>
      <button class="btn btn-secondary" type="button" @click.prevent="close">{{ $t('general.close') }}</button>
    </template>
  </Modal>
</template>
