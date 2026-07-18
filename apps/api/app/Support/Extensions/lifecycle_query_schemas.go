package extensionsruntime

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	queryregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/QueryRegistry"
)

const lifecycleQuerySchemaMaximumBytes = 1 << 20

func bindLifecycleQuerySchemas(
	extension extensions.Extension,
	publication queryregistry.Publication,
) (queryregistry.Publication, error) {
	bindings := make([]queryregistry.JSONResultSchemaBinding, 0, len(extension.Manifest.Queries))
	for _, query := range extension.Manifest.Queries {
		// handlerless legacy Query 仅 inspect/plan；不要求 package Schema 文件。
		if strings.TrimSpace(query.Handler) == "" {
			continue
		}
		file, err := exactLifecycleQuerySchemaFile(extension.Manifest.PackageFiles, query.ResultSchema)
		if err != nil {
			return queryregistry.Publication{}, fmt.Errorf("%w: query %s result Schema: %v",
				ErrLifecycleRegistryPublicationInvalid, query.ID, err)
		}
		body, err := readExactLifecycleQuerySchema(extension, file)
		if err != nil {
			return queryregistry.Publication{}, fmt.Errorf("%w: query %s result Schema: %v",
				ErrLifecycleRegistryPublicationInvalid, query.ID, err)
		}
		bindings = append(bindings, queryregistry.JSONResultSchemaBinding{
			QueryID: query.ID, ContractVersion: query.ContractVersion,
			PlanVersion: query.PlanVersion, ResultSchema: query.ResultSchema,
			Artifact: publication.Artifact, SchemaDigest: strings.ToLower(file.Digest), Schema: body,
		})
	}
	bound, err := queryregistry.BindResultSchemas(publication, bindings)
	if err != nil {
		return queryregistry.Publication{}, fmt.Errorf("%w: bind exact query result Schemas: %v",
			ErrLifecycleRegistryPublicationInvalid, err)
	}
	return bound, nil
}

func exactLifecycleQuerySchemaFile(
	files []extensions.ManifestPackageFile,
	reference string,
) (extensions.ManifestPackageFile, error) {
	reference = strings.TrimSpace(reference)
	var match *extensions.ManifestPackageFile
	for index := range files {
		file := &files[index]
		if file.Kind != "schema" ||
			(reference != file.Path && reference != file.ID+"@"+file.Version) {
			continue
		}
		if match != nil {
			return extensions.ManifestPackageFile{}, errors.New("multiple exact package Schema entries match")
		}
		value := *file
		match = &value
	}
	if match == nil || strings.TrimSpace(match.Path) == "" || len(match.Digest) != sha256.Size*2 {
		return extensions.ManifestPackageFile{}, errors.New("exact package Schema entry is missing")
	}
	return *match, nil
}

func readExactLifecycleQuerySchema(
	extension extensions.Extension,
	file extensions.ManifestPackageFile,
) ([]byte, error) {
	// 复用 Models 稳定 exact-digest reader：OpenRoot + SameFile + O_NONBLOCK +
	// LimitReader + digest 校验；拒绝 EvalSymlinks/os.ReadFile 的路径逃逸与无界读。
	body, _, err := extensions.ReadExactExtensionDigestFile(
		extension, file.Path, file.Digest, lifecycleQuerySchemaMaximumBytes, true,
	)
	if err != nil {
		return nil, err
	}
	return body, nil
}
