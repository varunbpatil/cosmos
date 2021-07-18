<template>

  <v-dialog max-width="800" v-model="dialog" scrollable>
    <!-- activator button -->
    <template v-slot:activator="{ on, attrs }">
      <v-row no-gutters justify="end">
        <v-btn v-bind="attrs" v-on="on" dark tile depressed color="indigo">
          <v-icon left>mdi-plus</v-icon>
          <span>NEW</span>
        </v-btn>
      </v-row>
    </template>

    <!-- create-sync form displayed within the dialog -->
    <v-card>
      <v-toolbar flat dark dense color="indigo darken-1">
        <v-toolbar-title>Create a new sync</v-toolbar-title>
        <v-spacer></v-spacer>
        <v-icon @click="dialog = false">mdi-close</v-icon>
      </v-toolbar>

      <v-card-text class="py-6">
        <v-text-field
          outlined
          color="indigo"
          label="Name"
          v-model.trim="sync.name"
          class="pt-3"
        ></v-text-field>

        <!--selection box for source endpoint-->
        <!--item-text prop controls what to display inside the selection box-->
        <!--item-value prop controls what value should be associated with a particular selection and what value is assigned to the v-model-->
        <v-autocomplete
          outlined
          :loading="!sourceEndpoints"
          :items="sourceEndpoints"
          item-text="name"
          item-value="id"
          v-model="sync.sourceEndpointID"
          label="Source endpoint"
          color="indigo"
          item-color="indigo"
          clearable
          class="pt-3"
        ></v-autocomplete>

        <!--selection box for destination endpoint-->
        <!--item-text prop controls what to display inside the selection box-->
        <!--item-value prop controls what value should be associated with a particular selection and what value is assigned to the v-model-->
        <v-autocomplete
          outlined
          :loading="!destinationEndpoints"
          :items="destinationEndpoints"
          item-text="name"
          item-value="id"
          v-model="sync.destinationEndpointID"
          label="Destination endpoint"
          color="indigo"
          item-color="indigo"
          clearable
          class="pt-3"
        ></v-autocomplete>

        <v-text-field
          outlined
          v-model.number="sync.scheduleInterval"
          label="Schedule interval"
          suffix="minutes"
          hint="Set the schedule interval in minutes"
          color="indigo"
          class="pt-3"
        ></v-text-field>

        <div v-if="form">
          <v-row v-for="(f, idx) in form.catalog" :key="idx" no-gutters class="mt-12">
            <v-col cols="12" md="5">
              <v-checkbox
                v-model="f.isStreamSelected"
                :label="f.streamName"
                class="py-0"
                color="indigo"
              ></v-checkbox>
            </v-col>

            <v-col cols="12" md="7">
              <v-row no-gutters>
                <v-autocomplete
                  outlined
                  v-model="f.selectedSyncMode"
                  label="Select sync mode"
                  :items="f.syncModes"
                  :item-text="(item) => item.join(' - ')"
                  return-object
                  color="indigo"
                  item-color="indigo"
                ></v-autocomplete>
              </v-row>
              <v-row no-gutters>
                <v-autocomplete
                  outlined
                  v-if="f.selectedSyncMode[0] === 'incremental'"
                  v-model="f.selectedCursorField"
                  label="Select cursor"
                  :items="f.cursorFields"
                  :item-text="(item) => item.join('.')"
                  return-object
                  color="indigo"
                  item-color="indigo"
                ></v-autocomplete>
              </v-row>
              <v-row no-gutters>
                <v-autocomplete
                  outlined
                  v-if="f.selectedSyncMode[1].endsWith('dedup')"
                  v-model="f.selectedPrimaryKey"
                  label="Select primary key"
                  :items="f.primaryKeys"
                  :item-text="(item) => item.join('.')"
                  multiple
                  return-object
                  color="indigo"
                  item-color="indigo"
                ></v-autocomplete>
              </v-row>
            </v-col>
          </v-row>
        </div>

        <div v-if="error" style="white-space: pre-line" class="text-body-1 red--text text--darken-2 mt-8">{{ error }}</div>
      </v-card-text>

      <v-card-actions>
        <v-spacer></v-spacer> <!-- This moves the button to the right -->
        <v-btn tile outlined color="indigo" class="body-2 font-weight-bold" :loading="loading" :disabled="!form" @click="createSync()">CREATE</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>

</template>


<script>
const _ = require('lodash')

export default {

  data() {
    return {
      sync: {
        name: "",
        sourceEndpointID: null, // the v-autocomplete component sets this to the appropriate value (see 'item-value' prop)
        destinationEndpointID: null, // the v-autocomplete component sets this to the appropriate value (see 'item-value' prop)
        scheduleInterval: null,
        config: null,
      },
      endpoints: [], // this has to be a non-null value for v-autocomplete to be rendered correctly.
      form: null,
      dialog: false,
      loading: false,
      error: null,
    }
  },

  computed: {

    sourceEndpoints() {
      return this.endpoints.filter(a => a.type === "source")
    },

    destinationEndpoints() {
      return this.endpoints.filter(a => a.type === "destination")
    },

  },

  watch: {

    // Clear form fields everytime the dialog opens.
    // Also, fetch all the connectors for the given endpointType.
    dialog: function(val) {
      if (val) {
        this.sync.name = ""
        this.sync.sourceEndpointID = null
        this.sync.destinationEndpointID = null
        this.sync.scheduleInterval = null
        this.sync.config = null
        this.endpoints = [] // this has to be an empty array, otherwise v-autocomplete will throw errors.
        this.form = null
        this.loading = false
        this.error = null

        this.$axios
          .get("/api/v1/endpoints")
          .then(response => {
            this.endpoints = response.data.endpoints
          })
      }
    },

    // When 'sync.sourceEndpointID' is set by the v-autocomplete field,
    // we need to fetch the catalog form for that sync.
    'sync.sourceEndpointID': function(val) {
      if (val && this.sync.destinationEndpointID) {
        this.getCreateForm()
      } else {
        // Whenever the source endpoint v-autocomplete field is "cleared", reset the form.
        this.form = null
        this.error = null
      }
    },

    'sync.destinationEndpointID': function(val) {
      if (val && this.sync.sourceEndpointID) {
        this.getCreateForm()
      } else {
        // Whenever the destination endpoint v-autocomplete field is "cleared", reset the form.
        this.form = null
        this.error = null
      }
    },

    "sync.scheduleInterval": function(val) {
      if (val === "") {
        this.sync.scheduleInterval = null
      }
    }
  },

  methods: {
    getCreateForm() {
      this.$axios
        .get(`/api/v1/endpoints/${this.sync.sourceEndpointID}/${this.sync.destinationEndpointID}/catalog-form`)
        .then(response => {
          this.form = response.data
        })
        .catch((error) => {
          if (error.response) {
            this.error = error.response.data.error
          }
        })
    },

    createSync() {
      // For error handling using axios, see https://gist.github.com/fgilio/230ccd514e9381fafa51608fcf137253
      this.loading = true
      this.error = null

      // We first make a deep copy of the "sync" so that it doesn't get changed from underneath us.
      let _sync = _.cloneDeep(this.sync)
      _sync.config = _.cloneDeep(this.form)

      this.$axios
        .post("/api/v1/syncs", _sync)
        .then(() => {
          // Close the dialog.
          this.dialog = false
          this.$emit("create", _sync.name)
        })
        .catch((error) => {
          if (error.response) {
            this.error = error.response.data.error
          }
        })
        .finally(() => {
          this.loading = false
        })
    }
  }
}
</script>
