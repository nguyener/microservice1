package main

import (
    tf "miroservice1.nguyener.com/service/service"
    "github.com/gin-gonic/gin"
    "github.com/gocql/gocql"
    "net/http"
    "fmt"
    "time"
)


func main() {
    cluster := gocql.NewCluster("10.120.136.124")
    cluster.Keyspace = tf.KEY_SPACE
    cluster.ProtoVersion = 3
    cluster.Consistency = gocql.Quorum
    session,_ := cluster.CreateSession() 
    
    tf.Conn = session
    
    if err := tf.BuildTables(); err != nil {
        fmt.Printf("Failed to setup tables.  Error:", err)
        return
    } 

    
    r := gin.Default()

    //tempates and template's commits (version)
    r.GET("/templates", GetTemplates)
    r.GET("/templates/:template_name", GetTemplateCommits)
    r.POST("/templates", CreateTemplate)
    r.DELETE("/templates/:template_name", DeleteTemplate)
    
   
    
    //template resources (file or release)
    //r.GET("/template_resource/:template_hash/:files_or_releases", GetTemplateResources)
    //r.GET("/templates/:template_hash/:files_or_releases/:resource_name", GetTemplateResource)
    r.POST("/template_resource/:template_hash/:file_or_release", CreateTemplateResource)
    r.PUT("/template_resource/:template_hash/:file_or_release/:resource_name", ModifyTemplateResource)
    r.DELETE("/template_resource/:template_hash/:file_or_release/:resource_name", DeleteTemplateResource)
    
    //devices
    r.GET("/devices", GetAllDevices)
    r.GET("/devices/:device_name", GetDevice)
    r.POST("/devices", CreateDevice)
    r.PUT("/devices/:device_name", UpdateDevice)
    r.DELETE("/devices/:device_name", DeleteDevice)
    r.Run(":8080")


//    var template tf.Template
//    fmt.Println(template.Template_name=="")
}

/***********************************************************************************************************
 * REST APIs
 **********************************************************************************************************/
// Get all templates (versions and duplicates due to multiple release role)
func GetTemplates(c *gin.Context){
    var template tf.Template
   
    c.JSON(http.StatusOK, template.GetTemplates())
}

//create a new templates
func CreateTemplate(c *gin.Context){

    type TemplatePayload struct {
        Template_name  string `json:"template_name" binding:"required"`
        Commit_message string `json:"commit_message" binding:"required"`
        Files []tf.File `json:"files"`
    }

    var payload TemplatePayload
    c.Bind(&payload)

    var template tf.Template

    if &payload.Template_name == nil {
        c.String(http.StatusOK, "Template name is not provided")
    } else if &payload.Commit_message == nil {
        c.String(http.StatusOK, "Commit message is not provided")
    } else {
        template.Template_hash = tf.GenerateHashString()
        template.Template_name = payload.Template_name
        template.Release_info = tf.ReleaseInfo{"NULL", ""}
        template.Commit_info = tf.CommitInfo{"nguyener", time.Now().String(), payload.Commit_message, ""}
        template.Files = payload.Files

        if err := template.CreateTemplate(); err != nil {
            c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to create new template.  Error: %s", err))
        }else{
            c.String(http.StatusOK, "A new template has been created successfully")
        }
    }
}


func DeleteTemplateResource(c *gin.Context){
    template_hash := c.Params.ByName("template_hash")
    resource_type := c.Params.ByName("file_or_release")
    resource_name := c.Params.ByName("resource_name")
    if resource_type != "files" && resource_type != "release_info" {
        c.String(http.StatusBadRequest, "Unsupport resource type")
    }else{
        var template tf.Template
        template.Template_hash = template_hash

        if resource_type == "release_info" {
            if err := template.DeleteTemplateRelease(resource_name); err != nil {
                c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to delete template release.  Error %s", err))
            } else {
                c.String(http.StatusOK, "A release has been delete successfully")
            }
        }else{
            fmt.Println("Deleting file " + resource_name)

            if err := template.DeleteTemplateFile(resource_name); err != nil {
                c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to delete template fil.  Error: %s", err))
            } else {
                c.String(http.StatusOK, "A template files has been deleted successfully")
            }

        }
    }
}

func ModifyTemplateResource(c *gin.Context){
    template_hash := c.Params.ByName("template_hash")
    resource_type := c.Params.ByName("file_or_release")
    resource_name := c.Params.ByName("resource_name")

    if resource_type != "files" && resource_type != "release_info" {
        c.String(http.StatusBadRequest, "Unsupport method")
    } else {
        var template tf.Template
        template.Template_hash = template_hash

        if resource_type == "release_info" {
            var payload tf.ReleaseInfo
            c.Bind(&payload)
            if err := template.ModifyTemplateRelease(resource_name, payload); err != nil {
                c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to modify template release. Error: %s", err))
            } else {
                c.String(http.StatusOK, "A release has been modified successfully")
            }
        }else{
            var payload tf.File
            c.Bind(&payload)
            fmt.Println(resource_name)
            fmt.Println(payload)
            if err := template.ModifyTemplateFile(resource_name, payload); err != nil {
                c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to modify template file.  Error: %s", err))
            } else {
                c.String(http.StatusOK, "A template file has been modified successfully")
            }

        }
    }
}


func CreateTemplateResource(c *gin.Context){
    template_hash := c.Params.ByName("template_hash")
    resource_type := c.Params.ByName("file_or_release")

    //DEBUG
    fmt.Println(template_hash)
    fmt.Println(resource_type)

    if resource_type != "files" && resource_type != "release_info" {
        c.String(http.StatusBadRequest, "Unsupport method")
    } else {
        var template tf.Template
        template.Template_hash = template_hash

        if resource_type == "release_info" {
            var payload tf.ReleaseInfo
            c.Bind(&payload)
            if err := template.CreateTemplateRelease( payload); err != nil {
                c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to created template release.  Error %s", err))
            } else {
                c.String(http.StatusOK, "A release has been created successfully")
            }
        }else{
            var payload tf.File
            c.Bind(&payload)

            if err := template.AddTemplateFile(payload); err != nil {
                c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to add template file.  Error: %s", err))
            } else {
                c.String(http.StatusOK, "A template file has been add successfully")
            }

        }
    }
}

// Get all commits (versions) of a specified template
//(inclues duplicates due to multple release role.  caller needs to process it properly)
func GetTemplateCommits(c *gin.Context){
    template_name := c.Params.ByName("template_name")

    var template tf.Template
    templates := template.GetTemplates()
    var templates_by_name []tf.Template
    for _, t := range templates {
        if t.Template_name == template_name {
            templates_by_name = append(templates_by_name, t)
        }
    }

    c.JSON(http.StatusOK,templates_by_name)

}




//delete all commits (versions) of a specified templates
func DeleteTemplate(c *gin.Context){
    /*
    template_name := c.Params.ByName("template_name")

    //Verify that none of commits are being used (which mapped to a device)
    iter := Conn.Query(QUERY_STR_GET_TEMPLATE_TO_DELETE).Iter()
    used := false
    var template tf.Template
    var template_ids []string
    for iter.Scan(&template.Template_id, &template.Template_name, &template.Release_info){
        if template.Template_name == template_name {
            template_ids = append(template_ids, template.Template_id)
            if &template.Release_info != nil {
                used = true
                break
            }
        }
    }

    if &template_ids == nil {
        c.String(http.StatusNotFound, fmt.Sprintf("Template %s does not exist", template_name))
    }else{
        if used {
            c.String(http.StatusConflict, fmt.Sprintf("Template %s is currently mapped to a device", template_name))
        }else{

            query_str :=  fmt.Sprintf(QUERY_STR_DELETE_TEMPLATE_ROWS, strings.Join(template_ids, ","))

            if err := Conn.Query(query_str).Exec(); err != nil {
                c.String(http.StatusInternalServerError, "Failed to delete template " + template_name)
            }else{
                c.String(http.StatusOK, "Successfully deleted template " + template_name)
            }
       }
   }
*/
}



func GetAllDevices(c *gin.Context){
    var device tf.Device  
    devices := device.GetAllDevices()
    c.JSON(http.StatusOK, devices)
}

func GetDevice(c *gin.Context){
    device_name := c.Params.ByName("device_name")
    var device tf.Device
    device.Device_name = device_name
    if device, err := device.GetDevice(); err != nil {
        c.String(http.StatusInternalServerError,fmt.Sprintf("Failed to get device %s", device_name))
    } else {
        c.JSON(http.StatusOK, device)
    }
}



func CreateDevice(c *gin.Context){
    var payload tf.Device
    c.Bind(&payload)
    var device tf.Device
    
    if err := device.CreateDevice(payload); err != nil {
        c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to create a new  device.  Error: %s", err))
    } else {
        c.String(http.StatusOK, "A new device has been created successfully")
    }

}


func UpdateDevice(c *gin.Context){
    device_name := c.Params.ByName("device_name")
    var payload tf.Device
    c.Bind(&payload)
    var device tf.Device
    device.Device_name = device_name
    if err := device.UpdateDevice(payload); err != nil {
        c.String(http.StatusInternalServerError,fmt.Sprintf("Failed to update device %s.  Error: ", device_name, err))
    } else {
        c.String(http.StatusOK, fmt.Sprintf("Device %s has been updated successfully", device_name))
    }
}


func DeleteDevice(c *gin.Context){
    device_name := c.Params.ByName("device_name")
    var device tf.Device
    device.Device_name = device_name
    fmt.Println(device_name)
    if err := device.DeleteDevice(); err != nil {
        c.String(http.StatusInternalServerError,fmt.Sprintf("Failed to delete device %s.  Error: %s", device_name, err))
    } else {
        c.String(http.StatusOK, fmt.Sprintf("Device %s has been deleted successfully", device_name))
    }
}









