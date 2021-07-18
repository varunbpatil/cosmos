<template>
  <v-dialog
    v-model="dialog"
    :max-width="options.width"
    :style="{ zIndex: options.zIndex }"
    @keydown.esc="cancel"
  >
    <v-card>
      <v-toolbar dark :color="options.color" dense flat>
        <v-toolbar-title>{{ title }}</v-toolbar-title>
      </v-toolbar>
      <v-card-text
        v-show="!!message"
        class="text-body-1 pa-4 black--text"
        v-html="message"
      ></v-card-text>
      <v-card-actions class="pt-3">
        <v-spacer></v-spacer>
        <v-btn
          tile
          text
          v-if="!options.noconfirm"
          color="grey"
          class="body-2 font-weight-bold"
          @click.native="cancel"
          >NO, I'M SCARED</v-btn
        >
        <v-btn
          tile
          outlined
          color="indigo"
          class="body-2 font-weight-bold"
          @click.native="agree"
          >YES</v-btn
        >
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script>
  export default {
    name: "ConfirmationDialog",
    data() {
      return {
        dialog: false,
        resolve: null,
        reject: null,
        message: null,
        title: null,
        options: {
          color: "indigo darken-1",
          width: 600,
          zIndex: 200,
          noconfirm: false,
        },
      };
    },

    methods: {
      open(title, message, options) {
        this.dialog = true;
        this.title = title;
        this.message = message;
        this.options = Object.assign(this.options, options);
        return new Promise((resolve, reject) => {
          this.resolve = resolve;
          this.reject = reject;
        });
      },
      agree() {
        this.resolve(true);
        this.dialog = false;
      },
      cancel() {
        this.resolve(false);
        this.dialog = false;
      },
    },
  };
</script>
