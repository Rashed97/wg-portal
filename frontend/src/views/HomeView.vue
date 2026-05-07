<script setup>
import { authStore } from "@/stores/auth";
import { RouterLink } from "vue-router";

const auth = authStore()
</script>

<template>
  <div class="page-header">
    <h1>{{ $t('home.headline') }}</h1>
  </div>

  <p class="lead">{{ $t('home.abstract') }}</p>


  <div class="card border-secondary p-5" v-if="auth.IsAuthenticated">
    <h2 class="display-5">{{ $t('home.profiles.headline') }}</h2>
    <p class="lead">{{ $t('home.profiles.abstract') }}</p>
    <hr class="my-4">
    <p class="card-text">{{ $t('home.profiles.content') }}</p>
    <p class="lead">
      <RouterLink :to="{ name: 'profile' }" class="btn btn-primary btn-lg">{{ $t('home.profiles.button') }}</RouterLink>
    </p>
  </div>

  <div class="card border-secondary p-5 mt-4" v-if="auth.IsAuthenticated && auth.IsAdmin">
    <h2 class="display-5">{{ $t('home.admin.headline') }}</h2>
    <p class="lead">{{ $t('home.admin.abstract') }}</p>
    <hr class="my-4">
    <p class="card-text">{{ $t('home.admin.content') }}</p>
    <p class="lead">
      <RouterLink :to="{ name: 'interfaces' }" class="btn btn-primary btn-lg me-2">{{ $t('home.admin.button-admin') }}
      </RouterLink>
      <RouterLink :to="{ name: 'users' }" class="btn btn-primary btn-lg">{{ $t('home.admin.button-user') }}</RouterLink>
    </p>
  </div>

  <h3 class="mt-5">{{ $t('home.info-headline') }}</h3>

  <!-- Section headers — keeps the protocol grouping visible across both card-pair rows below -->
  <div class="row">
    <div class="col-lg-6">
      <h4 class="mt-3">{{ $t('home.wg.section-header') }}</h4>
    </div>
    <div class="col-lg-6">
      <h4 class="mt-3">{{ $t('home.awg.section-header') }}</h4>
    </div>
  </div>

  <!-- Installation row: WG-install + AWG-install. Each card uses h-100 so the
       Bootstrap default align-items: stretch on the row makes them equal height. -->
  <div class="row">
    <div class="col-lg-6 mb-3">
      <div class="card border-secondary h-100">
        <div class="card-header">{{ $t('home.wg.installation.box-header') }}</div>
        <div class="card-body d-flex flex-column">
          <h5 class="card-title">{{ $t('home.wg.installation.headline') }}</h5>
          <p class="card-text">{{ $t('home.wg.installation.content') }}</p>
          <a href="https://www.wireguard.com/install/" title="WireGuard Installation" target="_blank"
            rel="noopener noreferrer" class="mt-auto btn btn-primary btn-sm align-self-start">{{ $t('home.wg.installation.button') }}</a>
        </div>
      </div>
    </div>
    <div class="col-lg-6 mb-3">
      <div class="card border-secondary h-100">
        <div class="card-header">{{ $t('home.awg.installation.box-header') }}</div>
        <div class="card-body d-flex flex-column">
          <h5 class="card-title">{{ $t('home.awg.installation.headline') }}</h5>
          <p class="card-text">{{ $t('home.awg.installation.content') }}</p>
          <!-- Per-platform direct download buttons. We deliberately
               link to the native AmneziaWG clients (not the full
               AmneziaVPN suite) since this portal only manages AWG. -->
          <div class="mt-auto">
            <div class="btn-group btn-group-sm" role="group" :aria-label="$t('home.awg.installation.platforms-label')">
              <a href="https://apps.apple.com/app/amneziawg/id6478942365" target="_blank" rel="noopener noreferrer"
                 class="btn btn-outline-primary" title="iOS / iPadOS / macOS">
                <i class="fab fa-apple me-1"></i>iOS / macOS
              </a>
              <a href="https://play.google.com/store/apps/details?id=org.amnezia.awg" target="_blank" rel="noopener noreferrer"
                 class="btn btn-outline-primary" title="Android">
                <i class="fab fa-android me-1"></i>Android
              </a>
              <a href="https://github.com/amnezia-vpn/amneziawg-windows-client/releases" target="_blank" rel="noopener noreferrer"
                 class="btn btn-outline-primary" title="Windows">
                <i class="fab fa-windows me-1"></i>Windows
              </a>
            </div>
            <div class="mt-2">
              <a href="https://docs.amnezia.org/documentation/amnezia-wg/#native-amneziawg-clients" target="_blank"
                 rel="noopener noreferrer" class="small">
                {{ $t('home.awg.installation.button-other') }}
              </a>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>

  <!-- About row: same equal-height pattern. -->
  <div class="row">
    <div class="col-lg-6 mb-3">
      <div class="card border-secondary h-100">
        <div class="card-header">{{ $t('home.wg.about.box-header') }}</div>
        <div class="card-body d-flex flex-column">
          <h5 class="card-title">{{ $t('home.wg.about.headline') }}</h5>
          <p class="card-text">{{ $t('home.wg.about.content') }}</p>
          <a href="https://www.wireguard.com/" title="WireGuard" target="_blank" rel="noopener noreferrer"
            class="mt-auto btn btn-primary btn-sm align-self-start">{{ $t('home.wg.about.button') }}</a>
        </div>
      </div>
    </div>
    <div class="col-lg-6 mb-3">
      <div class="card border-secondary h-100">
        <div class="card-header">{{ $t('home.awg.about.box-header') }}</div>
        <div class="card-body d-flex flex-column">
          <h5 class="card-title">{{ $t('home.awg.about.headline') }}</h5>
          <p class="card-text">{{ $t('home.awg.about.content') }}</p>
          <a href="https://github.com/amnezia-vpn/amneziawg-go" title="AmneziaWG" target="_blank"
            rel="noopener noreferrer" class="mt-auto btn btn-primary btn-sm align-self-start">{{ $t('home.awg.about.button') }}</a>
        </div>
      </div>
    </div>
  </div>

  <!-- About-portal full-width row — its own height, separate from the protocol pairs above. -->
  <div class="row">
    <div class="col-12 mb-4">
      <div class="card border-secondary">
        <div class="card-header">{{ $t('home.about-portal.box-header') }}</div>
        <div class="card-body d-flex flex-column">
          <h5 class="card-title">{{ $t('home.about-portal.headline') }}</h5>
          <p class="card-text">{{ $t('home.about-portal.content') }}</p>
          <a href="https://wgportal.org/" title="WireGuard Portal" target="_blank"
            rel="noopener noreferrer" class="mt-auto btn btn-primary btn-sm align-self-start">{{ $t('home.about-portal.button') }}</a>
        </div>
      </div>
    </div>
  </div>
</template>
