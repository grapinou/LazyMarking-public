package data

type PreviewQCMRoutes struct {
	PreviewQCM          string
	PreviewLandscapeQCM string
}

var DefaultPreviewQCMRoutes = PreviewQCMRoutes{
	PreviewQCM:          "/dashboard/qcm/preview_user_qcm",
	PreviewLandscapeQCM: "/dashboard/qcm/preview_user_qcm_landscape",
}
