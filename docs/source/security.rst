.. _test-pod-privileges:

===================
Test Pod Privileges
===================
The test-operator follows security best practices by applying a restricted
security profile to all test pods by default. This section describes the
defaults and when you might need to change them.

Default Security Profile
========================

Every test pod runs with the following restrictions:

* **Seccomp**: A :code:`RuntimeDefault` profile filters system calls at the
  kernel level.

* **Privilege escalation**: Disabled processes cannot gain more privileges
  than their parent.

* **Root filesystem**: Read-only containers cannot modify their own binaries
  or write to unexpected locations.

* **Capabilities**: All capabilities are dropped.

* **SELinux level**: Not set by default. Only needed for privileged
  workflows that write to shared PVCs.

When to Use Privileged Mode
===========================

.. important::
   Leave :code:`privileged: false` unless your tests specifically require it.

Set :code:`privileged: true` on the CR spec when your tests need capabilities
that the default profile blocks. The two capabilities added in privileged mode
are:

* :code:`NET_ADMIN` allows modifying network configuration such as routing
  tables, interfaces, and firewall rules.

* :code:`NET_RAW` allows raw socket access for tools like :code:`ping` and
  :code:`tcpdump`.

Privileged mode is also required when Tempest installs additional RPMs at
runtime via :code:`extraRPMs`, since that needs a writable root filesystem.
