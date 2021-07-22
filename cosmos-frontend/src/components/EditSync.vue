<template>
  <div>
    <ConfirmationDialog ref="confirm" />

    <v-dialog max-width="800" v-model="dialog" scrollable>
      <!-- activator button -->
      <!--using same button for two activators - v-dialog and v-tooltip-->
      <!--see https://stackoverflow.com/a/55271109-->
      <template v-slot:activator="{ on: ondialog, attrs: attrsdialog }">
        <v-tooltip bottom>
          <template v-slot:activator="{ on: ontooltip, attrs: attrstooltip }">
            <v-btn icon large v-bind="{ ...attrsdialog, ...attrstooltip }" v-on="{ ...ondialog, ...ontooltip }" color="indigo" @click.stop.prevent>
              <v-icon>mdi-square-edit-outline</v-icon>
            </v-btn>
          </template>
          <span class="font-weight-medium">Edit Sync</span>
        </v-tooltip>
      </template>

      <!-- edit-endpoint form displayed within the dialog -->
      <v-card>
        <v-toolbar flat dark dense color="indigo darken-1">
          <v-toolbar-title>Edit sync</v-toolbar-title>
          <v-spacer></v-spacer>
          <v-icon @click="dialog = false">mdi-close</v-icon>
        </v-toolbar>

        <v-card-text class="py-6">
          <v-text-field
            outlined
            color="indigo"
            label="Name"
            v-model.trim="localSync.name"
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
            v-model="localSync.sourceEndpointID"
            label="Source endpoint"
            color="indigo"
            item-color="indigo"
            clearable
            disabled
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
            v-model="localSync.destinationEndpointID"
            label="Destination endpoint"
            color="indigo"
            item-color="indigo"
            clearable
            disabled
            class="pt-3"
          ></v-autocomplete>

          <v-text-field
            outlined
            v-model.number="localSync.scheduleInterval"
            label="Schedule interval"
            suffix="minutes"
            hint="Set the schedule interval in minutes"
            color="indigo"
            class="pt-3"
          ></v-text-field>

          <!--Basic Normalization-->
          <v-switch
            v-if="supportsNormalization(localSync.destinationEndpointID)"
            v-model="localSync.basicNormalization"
            label="Basic Normalization"
            inset
            hide-details
            class="pt-3"
            color="indigo"
          ></v-switch>

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
          <v-menu bottom right>
            <template v-slot:activator="{ on, attrs }">
              <v-btn icon v-bind="attrs" v-on="on">
                  <v-icon>mdi-dots-vertical</v-icon>
              </v-btn>
            </template>

            <v-list dense>
              <v-list-item style="cursor: pointer" v-for="(option, idx) in clearOptions" :key="idx">
                <v-list-item-content>
                  <v-list-item-title class="red--text text--darken-2 font-weight-medium text-body-1" @click="clear(idx)">{{ option.title }}</v-list-item-title>
                </v-list-item-content>
              </v-list-item>
            </v-list>
          </v-menu>

          <v-spacer></v-spacer> <!-- This moves the buttons to the right -->

          <v-btn tile text class="body-2 font-weight-bold" color="red darken-2" :loading="isLoading('delete')" :disabled="isDisabled('delete')" @click="deleteSync()">DELETE</v-btn>
          <v-btn tile outlined class="body-2 font-weight-bold" color="indigo" :loading="isLoading('save')" :disabled="isDisabled('save')" @click="saveSync()">SAVE</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script>
const _ = require('lodash')

export default {
  components: {
    ConfirmationDialog: () => import("./ConfirmationDialog"),
  },

  props: {
    sync: Object
  },

  data() {
    return {
      // Array and Object props are passed by reference. So any changes we make
      // directly to the prop will be visible to the parent. This violates the
      // one-way data flow requirement (https://vuejs.org/v2/guide/components-props.html#One-Way-Data-Flow).
      // To avoid that, we make a deep copy of the prop.
      localSync: _.cloneDeep(this.sync),
      dialog: false,
      loading: null,
      error: null,
      endpoints: [],
      form: null,
      clearOptions: [
        {
          title: "clear incremental state",
          handler: this.clearIncrementalState
        },
        {
          title: "clear destination data",
          handler: this.clearDestinationData
        }
      ],
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
    // Reset form fields to the original value everytime the dialog opens.
    dialog: function(val) {
      if (val) {
        this.localSync = _.cloneDeep(this.sync)
        this.loading = null
        this.error = null
        this.endpoints = [],
        this.form = null,

        // Even though the v-autocomplete field for the endpoint selection box is disabled,
        // we still need to get the endpoints so that the endpoint name is displayed within the box.
        this.$axios
          .get("/api/v1/endpoints")
          .then(response => {
            this.endpoints = response.data.endpoints
          })

        this.$axios
          .get(`/api/v1/syncs/${this.localSync.id}/edit-form`)
          .then(response => {
            this.form = response.data
          })
      }
    },

    "localSync.scheduleInterval": function(val) {
      if (val === "") {
        this.localSync.scheduleInterval = null
      }
    }
  },

  methods: {
    isLoading(name) {
      return this.loading === name
    },

    isDisabled(name) {
      return this.loading && this.loading !== name
    },

    async deleteSync() {
      if (
          await this.$refs.confirm.open(
            "Confirm", "Are you sure you want to delete this sync?"
          )
      ) {
        // For error handling using axios, see https://gist.github.com/fgilio/230ccd514e9381fafa51608fcf137253
        this.loading = "delete"
        this.error = null

        this.$axios
          .delete("/api/v1/syncs/" + this.localSync.id)
          .then(() => {
            // Close the dialog.
            this.dialog = false
            this.$emit("delete", this.localSync.name)
          })
          .catch((error) => {
            if (error.response) {
              /*
               * The request was made and the server responded with a
               * status code that falls out of the range of 2xx
               */
              console.log(error.response.data);
              console.log(error.response.status);
              console.log(error.response.headers);

              this.error = error.response.data.error
            } else if (error.request) {
              /*
               * The request was made but no response was received, `error.request`
               * is an instance of XMLHttpRequest in the browser and an instance
               * of http.ClientRequest in Node.js
               */
              console.log(error.request);
            } else {
              // Something happened in setting up the request and triggered an Error
              console.log('Error', error.message);
            }
            console.log(error.config);
          })
          .finally(() => {
            this.loading = null
          })
      }
    },

    saveSync() {
      // For error handling using axios, see https://gist.github.com/fgilio/230ccd514e9381fafa51608fcf137253
      this.loading = "save"
      this.error = null

      // We first make a deep copy of the "localEndpoint" so that it doesn't get changed from underneath us.
      let _sync = _.cloneDeep(this.localSync)
      _sync.config = _.cloneDeep(this.form)

      // The reason I have to cherrypick fields to update is because there are
      // some fields in Sync which are only updated internally (ex: state).
      this.$axios
        .patch(
          `/api/v1/syncs/${_sync.id}`,
          {
            name: _sync.name,
            config: _sync.config,
            scheduleInterval: _sync.scheduleInterval,
            basicNormalization: _sync.basicNormalization,
          }
        )
        .then(() => {
          // Close the dialog.
          this.dialog = false
          this.$emit("save", _sync.name)
        })
        .catch((error) => {
          if (error.response) {
            /*
             * The request was made and the server responded with a
             * status code that falls out of the range of 2xx
             */
            console.log(error.response.data);
            console.log(error.response.status);
            console.log(error.response.headers);

            this.error = error.response.data.error
          } else if (error.request) {
            /*
             * The request was made but no response was received, `error.request`
             * is an instance of XMLHttpRequest in the browser and an instance
             * of http.ClientRequest in Node.js
             */
            console.log(error.request);
          } else {
            // Something happened in setting up the request and triggered an Error
            console.log('Error', error.message);
          }
          console.log(error.config);
        })
        .finally(() => {
          this.loading = null
        })
    },

    async clearIncrementalState() {
      if (
          await this.$refs.confirm.open(
            "Confirm", "Are you sure you want to clear the incremental state for this sync?"
          )
      ) {
        this.loading = "clearIncrementalState"
        this.error = null

        this.$axios
          .patch(`/api/v1/syncs/${this.localSync.id}`, {state: {}})
          .then(() => {
            // Close the dialog.
            this.dialog = false
            this.$emit("clearIncrementalState", this.localSync.name)
          })
          .catch((error) => {
            if (error.response) {
              /*
               * The request was made and the server responded with a
               * status code that falls out of the range of 2xx
               */
              console.log(error.response.data);
              console.log(error.response.status);
              console.log(error.response.headers);

              this.error = error.response.data.error
            } else if (error.request) {
              /*
               * The request was made but no response was received, `error.request`
               * is an instance of XMLHttpRequest in the browser and an instance
               * of http.ClientRequest in Node.js
               */
              console.log(error.request);
            } else {
              // Something happened in setting up the request and triggered an Error
              console.log('Error', error.message);
            }
            console.log(error.config);
          })
          .finally(() => {
            this.loading = null
          })
      }
    },

    async clearDestinationData() {
      if (
          await this.$refs.confirm.open(
            "Confirm", "Are you sure you want to clear all destination data for this sync?"
          )
      ) {
        this.loading = "clearDestinationData"
        this.error = null

        this.$axios
          .post(`/api/v1/syncs/${this.localSync.id}/sync-now`, {wipeDestination: true})
          .then(() => {
            // Close the dialog.
            this.dialog = false
            this.$emit("clearDestinationData", this.localSync.name)
          })
          .catch((error) => {
            if (error.response) {
              /*
               * The request was made and the server responded with a
               * status code that falls out of the range of 2xx
               */
              console.log(error.response.data);
              console.log(error.response.status);
              console.log(error.response.headers);

              this.error = "Cannot clear destination data. " + error.response.data.error
            } else if (error.request) {
              /*
               * The request was made but no response was received, `error.request`
               * is an instance of XMLHttpRequest in the browser and an instance
               * of http.ClientRequest in Node.js
               */
              console.log(error.request);
            } else {
              // Something happened in setting up the request and triggered an Error
              console.log('Error', error.message);
            }
            console.log(error.config);
          })
          .finally(() => {
            this.loading = null
          })
      }
    },

    clear(idx) {
      this.clearOptions[idx].handler()
    },

    supportsNormalization(destEndpointID) {
      if (!destEndpointID || this.destinationEndpoints.length == 0) {
        return false
      }
      let index = this.destinationEndpoints.findIndex(obj => obj.id == destEndpointID)
      return this.destinationEndpoints[index].connector.spec.spec.supportsNormalization
    }
  }
}
</script>
