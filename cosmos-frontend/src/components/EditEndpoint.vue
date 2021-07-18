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
          <span class="font-weight-medium">Edit Endpoint</span>
        </v-tooltip>
      </template>

      <!-- edit-endpoint form displayed within the dialog -->
      <v-card>
        <v-toolbar flat dark dense color="indigo darken-1">
          <v-toolbar-title>Edit endpoint</v-toolbar-title>
          <v-spacer></v-spacer>
          <v-icon @click="dialog = false">mdi-close</v-icon>
        </v-toolbar>

        <v-card-text class="py-6">
          <!--endpoint name-->
          <v-text-field
            outlined
            color="indigo"
            label="Name"
            v-model.trim="localEndpoint.name"
            class="pt-3"
          ></v-text-field>

          <!--Don't allow changing the connector (the autocomplete form below is disabled)-->
          <v-autocomplete
            outlined
            :loading="!connectors"
            :items="connectors"
            item-text="name"
            item-value="id"
            v-model="localEndpoint.connectorID"
            :label="`${capitalize(localEndpoint.type)} connector`"
            color="indigo"
            item-color="indigo"
            clearable
            disabled
            class="pt-3"
          ></v-autocomplete>

          <div v-if="form">
            <div v-for="(f, idx) in form.spec" :key="idx">
              <!--display all text fields in the configuration form as a text field-->
              <v-text-field
                outlined
                v-if="f.type === 'string' && !f.enum && dependencySatisfied(f, form)"
                :label="(f.title || f.path.filter(a => a.match(/<<\d+>>/g) === null).join(' / ')) + (f.required ? '*' : '')"
                :placeholder="f.examples ? f.examples.toString() : ''"
                :hint="f.description || ''"
                v-model.trim="f.value"
                :type="f.secret ? 'password' : ''"
                color="indigo"
                class="pt-3"
              >
                <!--This is to parse html content in hint-->
                <template v-slot:message="{message, key}">
                  <div v-html="message" :key="key"></div>
                </template>
              </v-text-field>

              <!--display all number fields in the configuration form as a text field with a number rule-->
              <v-text-field
                outlined
                v-if="(f.type === 'number' || f.type === 'integer') && !f.enum && dependencySatisfied(f, form)"
                :label="(f.title || f.path.filter(a => a.match(/<<\d+>>/g) === null).join(' / ')) + (f.required ? '*' : '')"
                :placeholder="f.examples ? f.examples.toString() : ''"
                :hint="f.description || ''"
                v-model.number="f.value"
                :rules="rules"
                :type="f.secret ? 'password' : ''"
                color="indigo"
                class="pt-3"
              >
                <!--This is to parse html content in hint-->
                <template v-slot:message="{message, key}">
                  <div v-html="message" :key="key"></div>
                </template>
              </v-text-field>

              <!--display enum and array of enum-->
              <v-select
                outlined
                v-if="f.enum && dependencySatisfied(f, form)"
                :items="f.enum"
                :menu-props="{ offsetY: true }"
                :label="(f.title || f.path.filter(a => a.match(/<<\d+>>/g) === null).join(' / ')) + (f.required ? '*' : '')"
                :placeholder="f.examples ? f.examples.toString() : ''"
                :hint="f.description || ''"
                v-model="f.value"
                :multiple="f.multiple"
                color="indigo"
                item-color="indigo"
                class="pt-3"
              >
                <!--This is to parse html content in hint-->
                <template v-slot:message="{message, key}">
                  <div v-html="message" :key="key"></div>
                </template>
              </v-select>

              <!--we currently have no handling for non-enum arrays. i.e, arrays which take arbitrary user input-->

              <!--display all boolean fields in the configuration form as a checkbox-->
              <v-checkbox
                v-if="f.type === 'boolean' && dependencySatisfied(f, form)"
                :label="(f.title || f.path.filter(a => a.match(/<<\d+>>/g) === null).join(' / ')) + (f.required ? '*' : '')"
                :hint="f.description || ''"
                v-model="f.value"
                color="indigo"
                class="pb-3"
              >
                <!--This is to parse html content in hint-->
                <template v-slot:message="{message, key}">
                  <div v-html="message" :key="key"></div>
                </template>
              </v-checkbox>
            </div>
          </div>

          <div v-if="error" style="white-space: pre-line" class="text-body-1 red--text text--darken-2 mt-8">{{ error }}</div>
        </v-card-text>

        <v-card-actions>
          <v-spacer></v-spacer> <!-- This moves the buttons to the right -->
          <v-btn tile text class="body-2 font-weight-bold" color="red darken-2" :loading="loadingDelete" :disabled="disableDelete" @click="deleteEndpoint()">DELETE</v-btn>
          <v-btn tile outlined class="body-2 font-weight-bold" color="indigo" :loading="loadingSave" :disabled="disableSave" @click="saveEndpoint()">SAVE</v-btn>
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
    endpoint: Object
  },

  data() {
    return {
      // Array and Object props are passed by reference. So any changes we make
      // directly to the prop will be visible to the parent. This violates the
      // one-way data flow requirement (https://vuejs.org/v2/guide/components-props.html#One-Way-Data-Flow).
      // To avoid that, we make a deep copy of the prop.
      localEndpoint: _.cloneDeep(this.endpoint),
      dialog: false,
      loadingDelete: false, // loading indicator on delete button.
      loadingSave: false,   // loading indicator on save button.
      disableDelete: false, // disables the delete button when save is running.
      disableSave: false,   // disables the save button when delete is running.
      error: null,

      connectors: [],
      form: null,
      rules: [
        v => /^[-+]?\d+[.]?\d*$/.test(v) || 'This field only accepts numbers'
      ]
    }
  },

  watch: {
    // Reset form fields to the original value everytime the dialog opens.
    dialog: function(val) {
      if (val) {
        this.localEndpoint = _.cloneDeep(this.endpoint)
        this.loadingDelete = false
        this.loadingSave = false
        this.disableDelete = false
        this.disableSave = false
        this.error = null
        this.connectors = [],
        this.form = null,

        // Even though the v-autocomplete field for the connector selection box is disabled,
        // we still need to get the connectors so that the connector name is displayed within the box.
        this.$axios
          .get("/api/v1/connectors?type=" + this.localEndpoint.type)
          .then(response => {
            this.connectors = response.data.connectors
          })

        this.$axios
          .get(`/api/v1/endpoints/${this.localEndpoint.id}/edit-form`)
          .then(response => {
            this.form = response.data
          })
      }
    }
  },

  methods: {
    // Capitalize the first character of the given string.
    capitalize(s) {
      return s.charAt(0).toUpperCase() + s.slice(1)
    },

    dependencySatisfied(field, form) {
      if (field.ignore) {
        return false
      }
      if (field.dependsOnIdx === null) {
        return true
      }
      if (field.dependsOnValue.includes(form.spec[field.dependsOnIdx].value)) {
        return this.dependencySatisfied(form.spec[field.dependsOnIdx], form)
      }
      return false
    },

    async deleteEndpoint() {
      if (
          await this.$refs.confirm.open(
            "Confirm", "Are you sure you want to delete this endpoint and all associated syncs?"
          )
      ) {
        // For error handling using axios, see https://gist.github.com/fgilio/230ccd514e9381fafa51608fcf137253
        this.loadingDelete = true
        this.disableSave = true // disable the other action (i.e, save).
        this.error = null

        this.$axios
          .delete("/api/v1/endpoints/" + this.localEndpoint.id)
          .then(() => {
            // Close the dialog.
            this.dialog = false
            this.$emit("delete", this.localEndpoint.name)
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

    saveEndpoint() {
      // For error handling using axios, see https://gist.github.com/fgilio/230ccd514e9381fafa51608fcf137253
      this.loadingSave = true
      this.disableDelete = true // disable the other action (i.e, delete).
      this.error = null

      // We first make a deep copy of the "localEndpoint" so that it doesn't get changed from underneath us.
      let _endpoint = _.cloneDeep(this.localEndpoint)
      _endpoint.config = _.cloneDeep(this.form)

      this.$axios
        .patch(`/api/v1/endpoints/${_endpoint.id}`, _endpoint)
        .then(() => {
          // Close the dialog.
          this.dialog = false
          this.$emit("save", _endpoint.name)
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
