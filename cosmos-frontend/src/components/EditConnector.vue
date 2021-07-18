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
          <span class="font-weight-medium">Edit Connector</span>
        </v-tooltip>
      </template>

      <!-- edit-connector form displayed within the dialog -->
      <v-card>
        <v-toolbar flat dark dense color="indigo darken-1">
          <v-toolbar-title>Edit connector</v-toolbar-title>
          <v-spacer></v-spacer>
          <v-icon @click="dialog = false">mdi-close</v-icon>
        </v-toolbar>

        <v-card-text class="py-6">
          <v-text-field
            outlined
            color="indigo"
            label="Name"
            v-model.trim="localConnector.name"
            class="pt-3"
          ></v-text-field>

          <v-text-field
            outlined
            color="indigo"
            label="Docker image name"
            v-model.trim="localConnector.dockerImageName"
            class="pt-3"
          ></v-text-field>

          <v-text-field
            outlined
            color="indigo"
            label="Docker image tag"
            v-model.trim="localConnector.dockerImageTag"
            class="pt-3"
          ></v-text-field>

          <v-select
            outlined
            v-if="localConnector.type === 'destination'"
            label="Destination type"
            v-model="localConnector.destinationType"
            :items="destinationTypes"
            color="indigo"
            item-color="indigo"
            class="pt-3"
          ></v-select>

          <div v-if="error" style="white-space: pre-line" class="text-body-1 red--text text--darken-2 mt-8">{{ error }}</div>
        </v-card-text>

        <v-card-actions>
          <v-spacer></v-spacer> <!-- This moves the buttons to the right -->
          <v-btn tile text class="body-2 font-weight-bold" color="red darken-2" :loading="loadingDelete" :disabled="disableDelete" @click="deleteConnector()">DELETE</v-btn>
          <v-btn tile outlined class="body-2 font-weight-bold" color="indigo" :loading="loadingSave" :disabled="disableSave" @click="saveConnector()">SAVE</v-btn>
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
    connector: Object
  },

  data() {
    return {
      // Array and Object props are passed by reference. So any changes we make
      // directly to the prop will be visible to the parent. This violates the
      // one-way data flow requirement (https://vuejs.org/v2/guide/components-props.html#One-Way-Data-Flow).
      // To avoid that, we make a deep copy of the prop.
      localConnector: _.cloneDeep(this.connector),
      destinationTypes: [],
      dialog: false,
      loadingDelete: false, // loading indicator on delete button.
      loadingSave: false,   // loading indicator on save button.
      disableDelete: false, // disables the delete button when save is running.
      disableSave: false,   // disables the save button when delete is running.
      error: null
    }
  },
  watch: {
    // Reset form fields to the original value everytime the dialog opens.
    dialog: function(val) {
      if (val) {
        this.localConnector = _.cloneDeep(this.connector)
        this.destinationTypes = []
        this.error = null
        this.loadingDelete = false
        this.loadingSave = false
        this.disableDelete = false
        this.disableSave = false

        this.$axios
          .get("/api/v1/connectors/destination-types")
          .then((response) => {
            this.destinationTypes = response.data
          })
      }
    }
  },
  methods: {
    async deleteConnector() {
      if (
          await this.$refs.confirm.open(
            "Confirm", "Are you sure you want to delete this connector and all associated endpoints and syncs?"
          )
      ) {
        // For error handling using axios, see https://gist.github.com/fgilio/230ccd514e9381fafa51608fcf137253
        this.loadingDelete = true
        this.disableSave = true // disable the other action (i.e, save).
        this.error = null

        this.$axios
          .delete("/api/v1/connectors/" + this.localConnector.id)
          .then(() => {
            // Close the dialog.
            this.dialog = false
            this.$emit("delete", this.localConnector.name)
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
            this.loadingDelete = false
            this.disableSave = false
          })
      }
    },

    saveConnector() {
      // For error handling using axios, see https://gist.github.com/fgilio/230ccd514e9381fafa51608fcf137253
      this.loadingSave = true
      this.disableDelete = true // disable the other action (i.e, delete).
      this.error = null

      this.$axios
        .patch(`/api/v1/connectors/${this.localConnector.id}`, this.localConnector)
        .then(() => {
          // Close the dialog.
          this.dialog = false
          this.$emit("save", this.localConnector.name)
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
          this.loadingSave = false
          this.disableDelete = false
        })
    }
  }
}
</script>
