# Changelog

## [0.1.3](https://github.com/validator-labs/validatorctl/compare/v0.1.2...v0.1.3) (2024-09-03)


### Features

* inline auth for MAAS ([#198](https://github.com/validator-labs/validatorctl/issues/198)) ([ed78617](https://github.com/validator-labs/validatorctl/commit/ed78617fe4386d8db2739f70af24f1cf90c474c0))
* support Azure plugin community gallery image rule ([#181](https://github.com/validator-labs/validatorctl/issues/181)) ([3d20725](https://github.com/validator-labs/validatorctl/commit/3d207256dbd269bc4c1c0f7f02ea67335147b631))


### Bug Fixes

* ensure ErrValidationFailed is returned for maas direct validation failures ([#179](https://github.com/validator-labs/validatorctl/issues/179)) ([0e0e7b9](https://github.com/validator-labs/validatorctl/commit/0e0e7b9ada5158fe258353e5df9daf968b0173c7))
* handle validation errors, result count mismatch ([#204](https://github.com/validator-labs/validatorctl/issues/204)) ([a2ea08b](https://github.com/validator-labs/validatorctl/commit/a2ea08b58d89f89d42deb92dda3cf1b16929c7d3))
* only require docker, kind when provisioning kind cluster ([#180](https://github.com/validator-labs/validatorctl/issues/180)) ([1bbdb0e](https://github.com/validator-labs/validatorctl/commit/1bbdb0e5505d023f2a84703f82f54020e22017a5))
* remove duplicate maas base values from template ([#182](https://github.com/validator-labs/validatorctl/issues/182)) ([cfa39a8](https://github.com/validator-labs/validatorctl/commit/cfa39a87423ff833f781d053fc57bbe4c750f329))
* support `validator rules check -f config.yaml` without all plugins defined ([#191](https://github.com/validator-labs/validatorctl/issues/191)) ([6829834](https://github.com/validator-labs/validatorctl/commit/6829834ee5323c0f6928b55a40b0112eeed53eaf))


### Other

* cleanup comment ([#176](https://github.com/validator-labs/validatorctl/issues/176)) ([f371927](https://github.com/validator-labs/validatorctl/commit/f3719278be6c023b79699dab3e898e73bdc6af84))


### Dependency Updates

* **deps:** update anchore/sbom-action action to v0.17.2 ([#184](https://github.com/validator-labs/validatorctl/issues/184)) ([00d3a5c](https://github.com/validator-labs/validatorctl/commit/00d3a5c47adcfd06f66f8d09799a2cf38fdeaf3a))
* **deps:** update github.com/validator-labs/validator-plugin-maas digest to e903cc7 ([#175](https://github.com/validator-labs/validatorctl/issues/175)) ([09c3ad1](https://github.com/validator-labs/validatorctl/commit/09c3ad12546849039fb9791d300384bd41a868c6))
* **deps:** update golang.org/x/exp digest to 9b4947d ([#190](https://github.com/validator-labs/validatorctl/issues/190)) ([8c427e9](https://github.com/validator-labs/validatorctl/commit/8c427e93ca0fd7ee6b1b76bcbe08d05bbafe23d9))
* **deps:** update module github.com/canonical/gomaasclient to v0.7.0 ([#197](https://github.com/validator-labs/validatorctl/issues/197)) ([f43a5ba](https://github.com/validator-labs/validatorctl/commit/f43a5baec0491721409657cb7bb0a69b8b31d6c5))
* **deps:** update module github.com/validator-labs/validator to v0.1.8 ([#120](https://github.com/validator-labs/validatorctl/issues/120)) ([fe587c6](https://github.com/validator-labs/validatorctl/commit/fe587c6e95d3a5f96bfb21ed874afcb1b8aef625))
* **deps:** update module github.com/vmware/govmomi to v0.42.0 ([#172](https://github.com/validator-labs/validatorctl/issues/172)) ([f0488c9](https://github.com/validator-labs/validatorctl/commit/f0488c940bfedcfb62256272f66bbd3fb26b1972))


### Refactoring

* move vsphere account under auth to match new plugin api ([#189](https://github.com/validator-labs/validatorctl/issues/189)) ([90352e1](https://github.com/validator-labs/validatorctl/commit/90352e1c81974558ca1ef96ad7af76e653f93b7a))

## [0.1.2](https://github.com/validator-labs/validatorctl/compare/v0.1.1...v0.1.2) (2024-08-19)


### Features

* add maas plugin ([#160](https://github.com/validator-labs/validatorctl/issues/160)) ([ab9f21a](https://github.com/validator-labs/validatorctl/commit/ab9f21a93f0d0634da0b984ff0bfc96039a7db98))
* allow selecting aws creds from filesystem ([#171](https://github.com/validator-labs/validatorctl/issues/171)) ([c3a714c](https://github.com/validator-labs/validatorctl/commit/c3a714c842a88e87712876034db3a276a05c73bd))
* allow specifying Azure cloud to connect to ([#170](https://github.com/validator-labs/validatorctl/issues/170)) ([6a4a704](https://github.com/validator-labs/validatorctl/commit/6a4a704452649e05253cd9ffdc0ded81e12545b4))
* read vCenter privileges from local file or editor ([#152](https://github.com/validator-labs/validatorctl/issues/152)) ([94ddd90](https://github.com/validator-labs/validatorctl/commit/94ddd90fb2ade3b3b7cba9551ca7ecb8f34dc7b3))
* set exit code 2 on validation failure; restore debug log file ([#150](https://github.com/validator-labs/validatorctl/issues/150)) ([2a3fe4d](https://github.com/validator-labs/validatorctl/commit/2a3fe4d4195a051f3779151ff58112bf2b26b109))
* support configuring oci validationType on a rule ([#161](https://github.com/validator-labs/validatorctl/issues/161)) ([8dfc501](https://github.com/validator-labs/validatorctl/commit/8dfc50176ab7e85740ddbbb2f0b70094391ea930))
* support direct oci validation of private registries ([#173](https://github.com/validator-labs/validatorctl/issues/173)) ([9cfeab9](https://github.com/validator-labs/validatorctl/commit/9cfeab99aea0301b0f0750d58dc85ae15357d29b))


### Bug Fixes

* correct TUI flow for `validator install -o --apply` ([#169](https://github.com/validator-labs/validatorctl/issues/169)) ([0912f6e](https://github.com/validator-labs/validatorctl/commit/0912f6e6dcae8e3c82e6967b8adb93b5dbad7c37))
* export creds for aws and azure direct check ([#167](https://github.com/validator-labs/validatorctl/issues/167)) ([5d569de](https://github.com/validator-labs/validatorctl/commit/5d569deb02bf2a442ae62e912cdff5416f379a9f))


### Dependency Updates

* **deps:** update anchore/sbom-action action to v0.17.1 ([#163](https://github.com/validator-labs/validatorctl/issues/163)) ([416d23c](https://github.com/validator-labs/validatorctl/commit/416d23cfe9fe43aa95dca044874f4504101561ae))
* **deps:** update github.com/validator-labs/validator-plugin-azure digest to b4687e5 ([#149](https://github.com/validator-labs/validatorctl/issues/149)) ([e7ab9a6](https://github.com/validator-labs/validatorctl/commit/e7ab9a637676d1640b2e60e5772f50f436a094e7))
* **deps:** update github.com/validator-labs/validator-plugin-vsphere digest to a93cb70 ([#147](https://github.com/validator-labs/validatorctl/issues/147)) ([79304b9](https://github.com/validator-labs/validatorctl/commit/79304b9087afd98348bb56ebb7716999d9d2e357))
* **deps:** update module github.com/vmware/govmomi to v0.40.0 ([#162](https://github.com/validator-labs/validatorctl/issues/162)) ([acf4a25](https://github.com/validator-labs/validatorctl/commit/acf4a25efb8ad90b1c1c5a1fef14688dafd96cab))


### Refactoring

* lazy configuration of oci auth and signature verification secrets ([#168](https://github.com/validator-labs/validatorctl/issues/168)) ([cc2c056](https://github.com/validator-labs/validatorctl/commit/cc2c05644d6877fd78d3d8713703bfa200b4b4a5))
* remove explicit TypeMetas; use vapi constants ([#154](https://github.com/validator-labs/validatorctl/issues/154)) ([28b321c](https://github.com/validator-labs/validatorctl/commit/28b321c1494ec616c1778e0356b04d9ab93600f1))

## [0.1.1](https://github.com/validator-labs/validatorctl/compare/v0.1.0...v0.1.1) (2024-08-08)


### Other

* clean up helpers for setting up oci network and vsphere validation rules ([#137](https://github.com/validator-labs/validatorctl/issues/137)) ([dbdee2f](https://github.com/validator-labs/validatorctl/commit/dbdee2f8fd22fc443ff67c96b36c301a39fe85e1))


### Docs

* update docs to reflect recent api changes ([#146](https://github.com/validator-labs/validatorctl/issues/146)) ([f2b3217](https://github.com/validator-labs/validatorctl/commit/f2b3217b086aa6d8d52ef71ea9dafd102c558fcb))


### Dependency Updates

* **deps:** update github.com/validator-labs/validator-plugin-azure digest to 862db62 ([#142](https://github.com/validator-labs/validatorctl/issues/142)) ([039e6f0](https://github.com/validator-labs/validatorctl/commit/039e6f0fc7143dd5ab58b69e4d27c2a25be08c88))
* **deps:** update github.com/validator-labs/validator-plugin-vsphere digest to d7deabd ([#143](https://github.com/validator-labs/validatorctl/issues/143)) ([72267ad](https://github.com/validator-labs/validatorctl/commit/72267adf955539b18878ab1edbc95d6121884561))
* **deps:** update module github.com/validator-labs/validator-plugin-oci to v0.1.0 ([#132](https://github.com/validator-labs/validatorctl/issues/132)) ([8862bb5](https://github.com/validator-labs/validatorctl/commit/8862bb50646701a095b1d6577f2cbdcdd44bf13a))


### Refactoring

* add rules subcommand & split out apply/check ([#144](https://github.com/validator-labs/validatorctl/issues/144)) ([e88bd71](https://github.com/validator-labs/validatorctl/commit/e88bd7130546eff1e75881984d2e05073ee78340))

## [0.1.0](https://github.com/validator-labs/validatorctl/compare/v0.0.6...v0.1.0) (2024-08-06)


### ⚠ BREAKING CHANGES

* split plugin rule configuration and installation into separate commands  ([#121](https://github.com/validator-labs/validatorctl/issues/121))

### Features

* add docs command; refactor to use embeddedfs pkg ([#116](https://github.com/validator-labs/validatorctl/issues/116)) ([dbe19c5](https://github.com/validator-labs/validatorctl/commit/dbe19c5b4d84d8142bb94956400b8625fb25a91f))
* read CA certs for network rules, add HTTPFileRules, AMIRules ([#117](https://github.com/validator-labs/validatorctl/issues/117)) ([0c4487f](https://github.com/validator-labs/validatorctl/commit/0c4487fc27d636ff21a791f5bfb75cd9576880a7))
* support direct rule evaluation with `validator check --direct` ([#127](https://github.com/validator-labs/validatorctl/issues/127)) ([f1fb0d6](https://github.com/validator-labs/validatorctl/commit/f1fb0d663a86da4798bd4f4a6462b6871b02fcd5))


### Docs

* added subcommands docs page ([#110](https://github.com/validator-labs/validatorctl/issues/110)) ([9fa23dc](https://github.com/validator-labs/validatorctl/commit/9fa23dcba0796a81859f595fc0c667dc557af993))


### Dependency Updates

* **deps:** update github.com/validator-labs/validator-plugin-azure digest to ba947e3 ([#134](https://github.com/validator-labs/validatorctl/issues/134)) ([2a1058d](https://github.com/validator-labs/validatorctl/commit/2a1058d5241d239eea763b44efbfd68b327a4fd3))
* **deps:** update github.com/validator-labs/validator-plugin-vsphere digest to 9b1f05b ([#135](https://github.com/validator-labs/validatorctl/issues/135)) ([253f328](https://github.com/validator-labs/validatorctl/commit/253f328f05efff65e9906239f6850d08b77a359b))


### Refactoring

* remove -s flag ([#126](https://github.com/validator-labs/validatorctl/issues/126)) ([9373e02](https://github.com/validator-labs/validatorctl/commit/9373e021e5a22cf1547cd57604df07ff725b86e3))
* simplify helm prompts ([#115](https://github.com/validator-labs/validatorctl/issues/115)) ([8ce75a1](https://github.com/validator-labs/validatorctl/commit/8ce75a1e763ff5dd459056fcd424409be2261a33))
* split plugin rule configuration and installation into separate commands  ([#121](https://github.com/validator-labs/validatorctl/issues/121)) ([6eaee77](https://github.com/validator-labs/validatorctl/commit/6eaee77fd8158ac2f43be8b1111175e1e9ef6b0f))

## [0.0.6](https://github.com/validator-labs/validatorctl/compare/v0.0.5...v0.0.6) (2024-07-26)


### Features

* Azure plugin - remove Palette presets, reading permission set files ([#97](https://github.com/validator-labs/validatorctl/issues/97)) ([95787db](https://github.com/validator-labs/validatorctl/commit/95787db0cd8e4edbb993ec22deeb17ff891c6d34))


### Other

* bump validator and plugin versions ([#106](https://github.com/validator-labs/validatorctl/issues/106)) ([a3863aa](https://github.com/validator-labs/validatorctl/commit/a3863aac993aa111d9aaaded20d0b54cc5cff866))

## [0.0.5](https://github.com/validator-labs/validatorctl/compare/v0.0.4...v0.0.5) (2024-07-24)


### Features

* add support for private custom image registries ([#83](https://github.com/validator-labs/validatorctl/issues/83)) ([ae91659](https://github.com/validator-labs/validatorctl/commit/ae91659286b9bab40f01a905d2279ee835c2abe8))
* support env vars in OCI secrets ([#88](https://github.com/validator-labs/validatorctl/issues/88)) ([584b3c7](https://github.com/validator-labs/validatorctl/commit/584b3c70a1a5fc94ce7c5a24dda731d76ff43f41))


### Bug Fixes

* ensure passwords in helm templates are quoted ([#96](https://github.com/validator-labs/validatorctl/issues/96)) ([f36383b](https://github.com/validator-labs/validatorctl/commit/f36383bc88e97aebf0c86c991bf2adcb1c4b0f42))


### Other

* omit EDITOR logs by default ([#76](https://github.com/validator-labs/validatorctl/issues/76)) ([b3ab7ec](https://github.com/validator-labs/validatorctl/commit/b3ab7ecb050429940d550f23270bca41ddc22ed1))


### Dependency Updates

* **deps:** update anchore/sbom-action action to v0.17.0 ([#75](https://github.com/validator-labs/validatorctl/issues/75)) ([caf800d](https://github.com/validator-labs/validatorctl/commit/caf800d3796d3ee5ea97ac793f18e73c2dd1c341))
* **deps:** update golang.org/x/exp digest to 8a7402a ([#89](https://github.com/validator-labs/validatorctl/issues/89)) ([3ffda87](https://github.com/validator-labs/validatorctl/commit/3ffda878f19b1836bce95d933f9b425f3c417eb3))
* **deps:** update golang.org/x/exp digest to e3f2596 ([#82](https://github.com/validator-labs/validatorctl/issues/82)) ([a89beb6](https://github.com/validator-labs/validatorctl/commit/a89beb6529aae14933959e0068a9627ecd455f95))
* **deps:** update module github.com/validator-labs/validator to v0.0.47 ([#92](https://github.com/validator-labs/validatorctl/issues/92)) ([8c359e1](https://github.com/validator-labs/validatorctl/commit/8c359e105a7b20c93b0006f6676fc93f67c54af6))
* **deps:** update module github.com/validator-labs/validator to v0.0.48 ([#94](https://github.com/validator-labs/validatorctl/issues/94)) ([743656d](https://github.com/validator-labs/validatorctl/commit/743656dc57da5a9b5811fd66ca2d71d92abac8d2))
* **deps:** update module github.com/validator-labs/validator-plugin-azure to v0.0.13 ([#79](https://github.com/validator-labs/validatorctl/issues/79)) ([0c2dff7](https://github.com/validator-labs/validatorctl/commit/0c2dff70689f37479bc4b7527e6fb427b4fcf14f))
* **deps:** update module github.com/validator-labs/validator-plugin-network to v0.0.18 ([#87](https://github.com/validator-labs/validatorctl/issues/87)) ([c69e355](https://github.com/validator-labs/validatorctl/commit/c69e355765539593bc09e8e84427563583efa879))
* **deps:** update module github.com/validator-labs/validator-plugin-network to v0.0.19 ([#95](https://github.com/validator-labs/validatorctl/issues/95)) ([e70433a](https://github.com/validator-labs/validatorctl/commit/e70433ab22eadadd82fde4bbe3fcaedfbafaa368))
* **deps:** update module github.com/validator-labs/validator-plugin-oci to v0.0.11 ([#90](https://github.com/validator-labs/validatorctl/issues/90)) ([a633962](https://github.com/validator-labs/validatorctl/commit/a6339628adb7efb19301fcb9458961d1267d7d81))
* **deps:** update module github.com/validator-labs/validator-plugin-vsphere to v0.0.27 ([#80](https://github.com/validator-labs/validatorctl/issues/80)) ([feb6360](https://github.com/validator-labs/validatorctl/commit/feb636043bca4c7b56330fb845387c3cf50f811c))
* **deps:** update module github.com/vmware/govmomi to v0.39.0 ([#93](https://github.com/validator-labs/validatorctl/issues/93)) ([2a01e95](https://github.com/validator-labs/validatorctl/commit/2a01e95b8d7cf52c420149426c6840ffa2fbcc18))
* **deps:** update softprops/action-gh-release digest to c062e08 ([#85](https://github.com/validator-labs/validatorctl/issues/85)) ([b8b5c62](https://github.com/validator-labs/validatorctl/commit/b8b5c626bf8c6b8078f1b9a0d193f4fb62688a7b))

## [0.0.4](https://github.com/validator-labs/validatorctl/compare/v0.0.3...v0.0.4) (2024-07-15)


### Features

* air-gapped support with hauler ([#74](https://github.com/validator-labs/validatorctl/issues/74)) ([aa3fd73](https://github.com/validator-labs/validatorctl/commit/aa3fd733817a7b0d971b5d4adfe1f710f99d7f49))


### Dependency Updates

* **deps:** update actions/setup-go digest to 0a12ed9 ([#72](https://github.com/validator-labs/validatorctl/issues/72)) ([7b6f978](https://github.com/validator-labs/validatorctl/commit/7b6f978baaeb496a88c9405d70fb03b5c2b9aa20))
* **deps:** update anchore/sbom-action action to v0.16.1 ([#71](https://github.com/validator-labs/validatorctl/issues/71)) ([347da36](https://github.com/validator-labs/validatorctl/commit/347da36864738dfe770889679105ba83347f050f))
* **deps:** update golang.org/x/exp digest to 46b0784 ([#67](https://github.com/validator-labs/validatorctl/issues/67)) ([89caf1c](https://github.com/validator-labs/validatorctl/commit/89caf1c40f1dce2642c17f65d5c4c6ae72653eb3))
* **deps:** update module github.com/validator-labs/validator to v0.0.46 ([#73](https://github.com/validator-labs/validatorctl/issues/73)) ([72897c2](https://github.com/validator-labs/validatorctl/commit/72897c201eaff04af1a03c2afc8cdff0845c68ad))
* **deps:** update module github.com/validator-labs/validator-plugin-aws to v0.1.1 ([#68](https://github.com/validator-labs/validatorctl/issues/68)) ([bba7058](https://github.com/validator-labs/validatorctl/commit/bba7058d6c7546bc281ecf5daac3372d694edf38))
* **deps:** update module github.com/validator-labs/validator-plugin-azure to v0.0.12 ([#70](https://github.com/validator-labs/validatorctl/issues/70)) ([e1bf9fa](https://github.com/validator-labs/validatorctl/commit/e1bf9fa80a38f9ced3b57b46860640cb68e9a6c3))
* **deps:** update module gopkg.in/yaml.v2 to v3 ([#61](https://github.com/validator-labs/validatorctl/issues/61)) ([8952d08](https://github.com/validator-labs/validatorctl/commit/8952d080157c4e155d1f851d535a542605cc299b))


### Refactoring

* enable revive and address all lints ([#69](https://github.com/validator-labs/validatorctl/issues/69)) ([b9c8df8](https://github.com/validator-labs/validatorctl/commit/b9c8df80f2c8c54c0fe10e38239281d05aadb290))

## [0.0.3](https://github.com/validator-labs/validatorctl/compare/v0.0.2...v0.0.3) (2024-06-27)


### Features

* add helpers to easily configure validator plugins ([#62](https://github.com/validator-labs/validatorctl/issues/62)) ([ae596d3](https://github.com/validator-labs/validatorctl/commit/ae596d349e755fed660373736498622e557ee051))


### Dependency Updates

* **deps:** update module github.com/vmware/govmomi to v0.38.0 ([#59](https://github.com/validator-labs/validatorctl/issues/59)) ([4e6ad15](https://github.com/validator-labs/validatorctl/commit/4e6ad1553b995ddf5b90e13e1d109d3b355d26c3))
* **deps:** update module gopkg.in/yaml.v2 to v3 ([#57](https://github.com/validator-labs/validatorctl/issues/57)) ([9698478](https://github.com/validator-labs/validatorctl/commit/96984785e059f2f35fe8aee7fe2d1ea7819d84fe))

## [0.0.2](https://github.com/validator-labs/validatorctl/compare/v0.0.1...v0.0.2) (2024-06-24)


### Features

* add support custom IAM role rules ([#50](https://github.com/validator-labs/validatorctl/issues/50)) ([912c0f3](https://github.com/validator-labs/validatorctl/commit/912c0f3fd491c17949f816552febf4197cc1f80d))


### Other

* remove vsphere palette oriented resources ([#34](https://github.com/validator-labs/validatorctl/issues/34)) ([9213c76](https://github.com/validator-labs/validatorctl/commit/9213c76510164d91f284863dd2790a9f9634ed46))


### Docs

* update README ([#58](https://github.com/validator-labs/validatorctl/issues/58)) ([38b469e](https://github.com/validator-labs/validatorctl/commit/38b469e0baa0e1c41fb73a8d75913c868667eefa))


### Dependency Updates

* **deps:** update github.com/spectrocloud-labs/prompts-tui digest to 3f0e83e ([#51](https://github.com/validator-labs/validatorctl/issues/51)) ([1aa0810](https://github.com/validator-labs/validatorctl/commit/1aa0810cb2ce67148af6e88da635a5e27f87dfbe))
* **deps:** update module gopkg.in/yaml.v2 to v3 ([#41](https://github.com/validator-labs/validatorctl/issues/41)) ([210f63e](https://github.com/validator-labs/validatorctl/commit/210f63e40d018a0f1e21f50a788c607b10975663))
* **deps:** update module gopkg.in/yaml.v2 to v3 ([#55](https://github.com/validator-labs/validatorctl/issues/55)) ([a0a13c0](https://github.com/validator-labs/validatorctl/commit/a0a13c038dfaec8cb7135fa23d5194f41240fc0c))
* **deps:** update module gopkg.in/yaml.v2 to v3 ([#56](https://github.com/validator-labs/validatorctl/issues/56)) ([8dc9ab2](https://github.com/validator-labs/validatorctl/commit/8dc9ab27e432209917ae63553d4f9b4e9fc0c1c2))
* **deps:** update softprops/action-gh-release digest to a74c6b7 ([#43](https://github.com/validator-labs/validatorctl/issues/43)) ([432891a](https://github.com/validator-labs/validatorctl/commit/432891a9b9232f6c66e88c0cf639913f6197d4f9))

## [0.0.1](https://github.com/validator-labs/validatorctl/compare/v0.0.1...v0.0.1) (2024-06-19)


### Features

* configure validatorctl ([#1](https://github.com/validator-labs/validatorctl/issues/1)) ([34285c6](https://github.com/validator-labs/validatorctl/commit/34285c60015173a261a35762a3ef206ee34ee794))
* ensure no binaries are embedded with validatorctl ([#31](https://github.com/validator-labs/validatorctl/issues/31)) ([02de3a5](https://github.com/validator-labs/validatorctl/commit/02de3a55d5a88aea6befc5958852a0f8585f9c83))
* only ask for role privilege user if running from admin account ([#16](https://github.com/validator-labs/validatorctl/issues/16)) ([6cdee2c](https://github.com/validator-labs/validatorctl/commit/6cdee2cc963c416cd8ad7ba90e73a2571b5fa2f6))


### Bug Fixes

* **deps:** update golang.org/x/exp digest to 7f521ea ([#18](https://github.com/validator-labs/validatorctl/issues/18)) ([c5e83cc](https://github.com/validator-labs/validatorctl/commit/c5e83cc2e4f4c85cc00a4f14ff2d6bbf08eb24d0))
* **deps:** update module github.com/spf13/cobra to v1.8.1 ([e6d64d5](https://github.com/validator-labs/validatorctl/commit/e6d64d5b5c77ee3ab2162079a55491fbdcae8252))
* **deps:** update module github.com/validator-labs/validator-plugin-network to v0.0.17 ([6b58097](https://github.com/validator-labs/validatorctl/commit/6b580976398b462d1df569332a720064f9c6f044))
* **deps:** update module gopkg.in/yaml.v2 to v3 ([#22](https://github.com/validator-labs/validatorctl/issues/22)) ([f56c29f](https://github.com/validator-labs/validatorctl/commit/f56c29ff81c380a3ef64c2bac1cb447ef7634f2b))
* **deps:** update module gopkg.in/yaml.v2 to v3 ([#23](https://github.com/validator-labs/validatorctl/issues/23)) ([c9b8708](https://github.com/validator-labs/validatorctl/commit/c9b870801dfa7bbdd0d6e4a48745a8859f40216f))
* **deps:** update module gopkg.in/yaml.v2 to v3 ([#35](https://github.com/validator-labs/validatorctl/issues/35)) ([3bfdf86](https://github.com/validator-labs/validatorctl/commit/3bfdf86a945f012168d01d7cdf283785c3469794))
* enable concurrent integration test execution ([#36](https://github.com/validator-labs/validatorctl/issues/36)) ([25a47de](https://github.com/validator-labs/validatorctl/commit/25a47de76359f92b635e1704f9dfeb52aea036ef))
* setup go for release builds ([a811556](https://github.com/validator-labs/validatorctl/commit/a8115568d5460fdade1ea5c057f5ec10c8e54f0d))


### Other

* release 0.0.1 ([5103eb7](https://github.com/validator-labs/validatorctl/commit/5103eb71337b104afc4d78ae3da776872fde2382))
* remove dead code ([#25](https://github.com/validator-labs/validatorctl/issues/25)) ([0631a09](https://github.com/validator-labs/validatorctl/commit/0631a0998a9c1e51610b6b3fb0cf4a77d3940024))
* remove logging to disk ([#30](https://github.com/validator-labs/validatorctl/issues/30)) ([c531b74](https://github.com/validator-labs/validatorctl/commit/c531b747dc9caf1a30f91e10c411ffd29f9ae491))


### Refactoring

* ensure int. tests succeed w/ a non-dev CLI version ([#40](https://github.com/validator-labs/validatorctl/issues/40)) ([19f0599](https://github.com/validator-labs/validatorctl/commit/19f0599763a2de9d831e97ebb1208bda99d03f56))
* use prompts-tui ReadCACert and file reader ([#14](https://github.com/validator-labs/validatorctl/issues/14)) ([d2bd299](https://github.com/validator-labs/validatorctl/commit/d2bd2998beb6f00bad0ed813af119242114b3986))
