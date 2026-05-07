
export function freshInterface() {
  return {
    Disabled: false,
    DisplayName: "",
    Identifier: "",
    CreateDefaultPeer: false,
    Mode: "server",
    Backend: "local",

    PublicKey: "",
    PrivateKey: "",

    ListenPort:  51820,
    Addresses: [],
    DnsStr: [],
    DnsSearch: [],

    Mtu: 0,
    FirewallMark: 0,
    RoutingTable: "",

    PreUp: "",
    PostUp: "",
    PreDown: "",
    PostDown: "",

    SaveConfig: false,

    // Peer defaults

    PeerDefNetwork: [],
    PeerDefDns: [],
    PeerDefDnsSearch: [],
    PeerDefEndpoint: "",
    PeerDefAllowedIPs: [],
    PeerDefMtu: 0,
    PeerDefPersistentKeepalive: 0,
    PeerDefFirewallMark: 0,
    PeerDefRoutingTable: "",
    PeerDefPreUp: "",
    PeerDefPostUp: "",
    PeerDefPreDown: "",
    PeerDefPostDown: "",

    // AmneziaWG V2 obfuscation parameters. Only meaningful when Backend ===
    // 'amneziawg'. Server side returns null for non-AWG interfaces; we keep
    // a fully-zeroed default here so the form bindings always have something
    // to point at and the API DTO round-trips cleanly.
    AmneziaWG: freshAmneziaWG(),

    TotalPeers: 0,
    EnabledPeers: 0,
    Filename: ""
  }
}

export function freshAmneziaWG() {
  return {
    Jc: 0, Jmin: 0, Jmax: 0,
    S1: 0, S2: 0, S3: 0, S4: 0,
    H1: 0, H2: 0, H3: 0, H4: 0,
    I1: "", I2: "", I3: "", I4: "", I5: "",
  }
}

export function freshPeer() {
  return {
    Identifier: "",
    DisplayName: "",
    UserIdentifier: "",
    UserDisplayName: "",
    InterfaceIdentifier: "",
    Disabled: false,
    ExpiresAt: null,
    Notes: "",

    Endpoint: {
      Value: "",
      Overridable: true,
    },
    EndpointPublicKey: {
      Value: "",
      Overridable: true,
    },
    AllowedIPs: {
      Value: [],
      Overridable: true,
    },
    ExtraAllowedIPs: [],
    PresharedKey: "",
    PersistentKeepalive: {
      Value: 0,
      Overridable: true,
    },

    PrivateKey: "",
    PublicKey: "",

    Mode: "client",

    Addresses: [],
    CheckAliveAddress: "",
    Dns: {
      Value: [],
      Overridable: true,
    },
    DnsSearch: {
      Value: [],
      Overridable: true,
    },
    Mtu: {
      Value: 0,
      Overridable: true,
    },
    FirewallMark: {
      Value: 0,
      Overridable: true,
    },
    RoutingTable: {
      Value: "",
      Overridable: true,
    },

    PreUp: {
      Value: "",
      Overridable: true,
    },
    PostUp: {
      Value: "",
      Overridable: true,
    },
    PreDown: {
      Value: "",
      Overridable: true,
    },
    PostDown: {
      Value: "",
      Overridable: true,
    },

    Filename: "",

    // Internal values
    IgnoreGlobalSettings: false,
    IsSelected: false
  }
}

export function freshUser() {
  return {
    Identifier: "",

    Email: "",
    AuthSources: ["db"],
    IsAdmin: false,

    Firstname: "",
    Lastname: "",
    Phone: "",
    Department: "",
    Notes: "",

    Password: "",

    Disabled: false,
    DisabledReason: "",
    Locked: false,
    LockedReason: "",

    ApiEnabled: false,

    PersistLocalChanges: false,

    PeerCount: 0,

    // Internal values
    IsSelected: false
  }
}

export function freshStats() {
  return {
    IsConnected: false,
    IsPingable: false,
    LastHandshake: null,
    LastPing: null,
    LastSessionStart: null,
    BytesTransmitted: 0,
    BytesReceived: 0,
    EndpointAddress: ""
  }
}