<template>
  <a-row justify="center">
    <a-col :xs="22" :sm="16" :md="12" :lg="8">
      <a-card>
        <a-typography-title :level="3" class="section-title">Register</a-typography-title>
        <a-form layout="vertical" @submit.prevent="submit">
          <a-form-item label="Username">
            <a-input v-model:value="form.username" placeholder="Username" />
          </a-form-item>
          <a-form-item label="Password">
            <a-input-password v-model:value="form.password" placeholder="Password" />
          </a-form-item>
          <a-form-item label="Confirm password">
            <a-input-password v-model:value="form.confirmPassword" placeholder="Confirm password" />
          </a-form-item>
          <a-button type="primary" block html-type="submit" :loading="loading">Create account</a-button>
        </a-form>
        <a-alert v-if="error" type="error" class="section-title" :message="error" show-icon />
        <RouterLink to="/login">Already have an account?</RouterLink>
      </a-card>
    </a-col>
  </a-row>
</template>

<script setup>
import { reactive, ref } from "vue";
import { RouterLink, useRouter } from "vue-router";
import { useAuthStore } from "../stores/auth";
import { message } from "ant-design-vue";

const form = reactive({
  username: "",
  password: "",
  confirmPassword: ""
});
const loading = ref(false);
const error = ref("");
const auth = useAuthStore();
const router = useRouter();

const submit = async () => {
  error.value = "";
  if (form.password !== form.confirmPassword) {
    error.value = "Passwords do not match";
    return;
  }
  loading.value = true;
  try {
    await auth.register({ username: form.username, password: form.password });
    message.success("Account created!");
    router.push("/friends");
  } catch (err) {
    error.value = err?.response?.data?.error || "Registration failed";
  } finally {
    loading.value = false;
  }
};
</script>
