<template>
  <v-container class="mt-4">

    <!--snackbar-->
    <v-snackbar v-model="snackbarToggle" :timeout="10000">
      <span class="font-weight-medium">{{ snackbarText }}</span>
      <template v-slot:action="{ attrs }">
        <v-btn color="yellow" text v-bind="attrs" @click="snackbarToggle = !snackbarToggle">CLOSE</v-btn>
      </template>
    </v-snackbar>

    <!-- create-new-connector dialog -->
    <CreateConnector :connectorType="connectorType" @create="snackbar('created', ...arguments)"></CreateConnector>

    <!-- list of connectors -->
    <v-card flat v-for="c in connectors" :key="c.id" class="mt-4">
      <v-card-text>
        <v-row>
          <v-col cols="12" sm="6" md="4" class="py-1">
            <div class="font-weight-medium indigo--text">Name</div>
            <div class="text-subtitle-1">{{ c.name }}</div>
          </v-col>
          <v-col cols="12" sm="6" md="5" class="py-1">
            <div class="font-weight-medium indigo--text">Image</div>
            <div class="text-subtitle-1">{{ c.dockerImageName }}</div>
          </v-col>
          <v-col cols="12" sm="6" md="2" class="py-1">
            <div class="font-weight-medium indigo--text">Version</div>
            <div class="text-subtitle-1">{{ c.dockerImageTag }}</div>
          </v-col>
          <!-- edit-connector dialog -->
          <v-col cols="12" sm="6" md="1" align-self="center" class="py-1">
            <EditConnector
              :connector="c"
              @delete="snackbar('deleted', ...arguments)"
              @save="snackbar('saved', ...arguments)"
            ></EditConnector>
          </v-col>
        </v-row>
      </v-card-text>
    </v-card>

  </v-container>
</template>

<script>
import EditConnector from '@/components/EditConnector'
import CreateConnector from '@/components/CreateConnector'

export default {
  components: {
    EditConnector,
    CreateConnector
  },

  data() {
    return {
      connectors: null,
      totalConnectors: null,
      snackbarToggle: false,
      snackbarText: null,
      intervalID: null,
    }
  },

  computed: {
    connectorType() {
      switch(this.$route.params.type) {
        case "sources":
          return "source"
        case "destinations":
          return "destination"
        default:
          return null
      }
    }
  },

  methods: {
    fetchConnectorsByType(type) {
      this.$axios
        .get("api/v1/connectors?type=" + type)
        .then(response => {
          this.connectors = response.data.connectors
          this.totalConnectors = response.data.totalConnectors
        })
    },

    // See https://stackoverflow.com/questions/53738919/emit-event-with-parameters-in-vue
    snackbar(action, target) {
      // Remove the previous snackbar text (if any).
      this.snackbarToggle = false
      // Wait until the snackbar is removed from the DOM before rendering the new snackbar text.
      // See https://vuejsdevelopers.com/2019/01/22/vue-what-is-next-tick/
      this.$nextTick(() => {
        this.snackbarText = "Successfully " + action + " " + target + " connector"
        this.snackbarToggle = true
      })
    }
  },

  mounted() {
    // If the connector type is not expected, redirect to /connectors.
    if (!this.connectorType) {
      this.$router.push("/connectors")
    }

    // First time connector fetch.
    this.fetchConnectorsByType(this.connectorType)

    // Subsequent connector fetches are done by setInterval.
    var v = this // Cannot access "this" directly inside setInterval.
    this.intervalID = setInterval(function() {
      v.fetchConnectorsByType(v.connectorType)
    }, 3000)
  },

  beforeDestroy() {
    clearInterval(this.intervalID)
  }
}
</script>
