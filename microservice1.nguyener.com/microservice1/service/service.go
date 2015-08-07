package templatefactory

import (
    "fmt"
    "errors"
    "time"
    "crypto/sha1"
    "strings"
    "github.com/gocql/gocql"
    "strconv"
)


const (
    KEY_SPACE                         = "templatefactory"
    TABLE_NAME_TEMPLATES              = "templates"
    TABLE_NAME_DEVICES                = "devices"

    QUERY_STR_CREATE_KEY_SPACE        = "CREATE KEYSPACE IF NOT EXISTS %s WITH REPLICATION = {'class': 'SimpleStrategy', 'replication_factor' :1}"
    QUERY_STR_CREATE_FILE_TYPE        = "CREATE TYPE IF NOT EXISTS file (file_name text, file_content text)"
    QUERY_STR_CREATE_COMMIT_INFO_TYPE = "CREATE TYPE IF NOT EXISTS commit_info (committer text, commit_date text, commit_message text, commit_action text)"
    QUERY_STR_CREATE_RELEASE_INFO_TYPE= "CREATE TYPE IF NOT EXISTS release_info (release_role text, firmware text)"
    QUERY_STR_CREATE_TEMPLATES_INDEX  = "CREATE INDEX templates_template_hash on templates(template_hash)"
    QUERY_STR_CREATE_DEVICES_TABLE    = "CREATE TABLE IF NOT EXISTS %s (device_name text primary key, template_name text, template_hash text, release_info frozen<release_info>)"
    QUERY_STR_CREATE_TEMPLATES_TABLE  = "CREATE TABLE IF NOT EXISTS %s (template_id uuid primary key,template_hash text, template_name text, commit_info frozen<commit_info>, release_info frozen<release_info>, files set<frozen<file>>)"



    //Insert queries
    QUERY_STR_INSERT_TEMPLATE         = "INSERT INTO templates (template_id, template_name, template_hash, commit_info, files, release_info) VALUES (%s, '%s', '%s', %s, {%s}, %s)"
    QUERY_STR_INSERT_DEVICE           = "INSERT INTO devices (device_name, template_name, template_hash, release_info) VALUES ('%s', '%s', '%s', %s)"

    //Select queries  
    QUERY_STR_GET_ALL_RELEASES        = "SELECT template_id, template_name, template_hash, release_info  FROM templates"
    QUERY_STR_GET_ALL_TEMPLATES_BASIC = "SELECT template_id, template_name, template_hash, commit_info FROM templates"
    QUERY_STR_GET_TEMPLATE_TO_DELETE  = "SELECT template_id, template_name, release_role FROM templates" 
    QUERY_STR_GET_TEMPLATE_FILES      = "SELECT files FROM templates WHERE template_hash='%s' LIMIT 1"
    QUERY_STR_GET_DEVICE_BY_NAME      = "SELECT device_name, template_name, template_hash, release_info from devices WHERE device_name='%s'"
    QUERY_STR_GET_ALL_DEVICES         = "SELECT device_name, template_name, template_hash, release_info from devices"
  
    //Delete queries
    QUERY_STR_DELETE_TEMPLATE_ROWS    = "DELETE FROM templates WHERE template_id in (%s)"
    QUERY_STR_DELETE_DEVICE   = "DELETE FROM devices WHERE device_name='%s'"
     
    //Update queries
    QUERY_STR_UPDATE_RELEASE_INFO     = "UPDATE templates SET release_info=%s WHERE template_id=%s"
    QUERY_STR_UPDATE_DEVICE           = "UPDATE devices SET template_name='%s', template_hash='%s', release_info=%s WHERE device_name='%s'"
   
)

var Conn *gocql.Session

func BuildTables() error {
    //create UDT for file, commit_info, release info
    if err := Conn.Query(QUERY_STR_CREATE_FILE_TYPE).Exec(); err != nil {
        return err
    }

    if err := Conn.Query(QUERY_STR_CREATE_RELEASE_INFO_TYPE).Exec(); err != nil {
        return err
    }

    if err := Conn.Query(QUERY_STR_CREATE_COMMIT_INFO_TYPE).Exec(); err != nil {
        return err
    }

    //create tables templates table if not exist
    query_str := fmt.Sprintf(QUERY_STR_CREATE_TEMPLATES_TABLE, TABLE_NAME_TEMPLATES)
    if err := Conn.Query(query_str).Exec(); err != nil {
        return err
    } 

    //create tables devices table if not exist
    query_str = fmt.Sprintf(QUERY_STR_CREATE_DEVICES_TABLE, TABLE_NAME_DEVICES)
    if err := Conn.Query(query_str).Exec(); err != nil {
        return err
    }

    return nil
}



 



/*********************************************************************************
 * Objects 
 ********************************************************************************/
type File struct {
    File_name    string  `cql:"file_name"`
    File_content string  `cql:"file_content"`
}

type ReleaseInfo struct {
    Release_role string `cql:"release_role"`
    Firmware     string `cql:"firmware"`
}

type CommitInfo struct {
    Committer      string  `cql:"committer"` 
    Commit_date    string  `cql:"commit_date"`
    Commit_message string  `cql:"commit_message"`
    Commit_action  string  `cql:"commit_action"`
}

type Template struct {
    Template_id       string            `cql:"template_id"`
    Template_name     string            `cql:"template_name"`
    Template_hash     string            `cql:"template_hash"`
    Commit_info       CommitInfo        `cql:"commit_info"`
    Release_info      ReleaseInfo       `cql:"release_info"`
    Files             []File            `cql:"files"` 
}
type Device struct {
        Device_name      string         `cql:"device_name"`
        Template_name    string         `cql:"template_name"`
        Template_hash    string         `cql:"template_hash"`
        Release_info     ReleaseInfo    `cql:"release_info"`
    }

func (f *File) toString() string {
    if f == nil {
        return "{}"
    } else {
        return fmt.Sprintf("{file_name:'%s', file_content:'%s'}", f.File_name, f.File_content)
    } 
}

func (c *CommitInfo) toString() string {
    if c == nil {
        return "null"
    } else {
        return fmt.Sprintf("{committer:'%s', commit_date:'%s', commit_message:'%s', commit_action:'%s'}", c.Committer, c.Commit_date, c.Commit_message, c.Commit_action)
    }
}


func (d *Device) toString() string {
    if d == nil {
        return "{}"
    } else {
        return fmt.Sprintf("{device_name:'%s', template_name:'%s', template_hash:'%s', release_role:'%s'}", d.Device_name, d.Template_name, d.Template_hash, d.Release_info.toString())
    }
}

func (r *ReleaseInfo) toString() string {
    if &r == nil || r.Release_role == "" {
        return "null"
    } else {
        return fmt.Sprintf("{release_role:'%s', firmware:'%s'}", r.Release_role, r.Firmware)
    } 
}

func (t *Template) GetTemplates() []Template {
    var template Template
    var templates []Template
    iter := Conn.Query(QUERY_STR_GET_ALL_TEMPLATES_BASIC).Iter()

    for iter.Scan(&template.Template_id, &template.Template_name, &template.Template_hash, &template.Commit_info){
        templates = append(templates,template)
    }
    return templates
}


// Create a new template record: a brand new template, a new template commit (version), or just a template release 
func (t *Template) CreateTemplate() error {
    
    //flatten files
    files_str := ""
    if &t.Files != nil && len(t.Files) > 0 {
        var files []string
        for _, file := range t.Files {
            files = append(files, file.toString())
        }
        files_str = strings.Join(files, ",")
    }

    query_str := fmt.Sprintf(QUERY_STR_INSERT_TEMPLATE, gocql.TimeUUID(), t.Template_name, t.Template_hash, t.Commit_info.toString(), files_str, t.Release_info.toString())

    //DEDBUG
    fmt.Println(query_str)
    err := Conn.Query(query_str).Exec()
    
    return err

}

//get all files from a template commit (version)
func (t *Template)  GetTemplateFiles() ([]File, error) {
    query_str := fmt.Sprintf(QUERY_STR_GET_TEMPLATE_FILES, t.Template_hash)
    fmt.Println(query_str)
    var files []File
    if err := Conn.Query(query_str).Scan(&files); err != nil {
        return nil, err
    } else {
        return files, nil
    } 
}
//Add files to specified template commit (version).  A new template commit (version) will generated.  Affect all releases under the original commit
func (t * Template) AddTemplateFile(new_file File ) error {
    
    files, _ := t.GetTemplateFiles()
    if len(files) > 0 {  
        for _, file := range files {
            if file.File_name == new_file.File_name {
                return errors.New("Could not add template files.  One or more specified files already not exists")                
             }
         }
    }

    files = append(files, new_file)    
    releases := t.GetTemplateCommitReleases()
    var err error
    for _, template := range releases {
       template.Template_hash = GenerateHashString()
       template.Commit_info = CommitInfo{"nguyener", time.Now().String(), "Add template files", "Add template files"}
       template.Files = files
       if err = template.CreateTemplate(); err != nil {
           return err
       }
    }
    return nil 
}            


//delete a file from specified template commit (version).  A new template commit (version) will generated.  Affect all releases under the original commit
func (t *Template) DeleteTemplateFile(file_name string) error {
    fmt.Println("about to delete template file")
    files,_ := t.GetTemplateFiles()
    fmt.Println("files len: " + strconv.Itoa(len(files)))

    index := -1
    for j, file := range files {
        if file.File_name == file_name {
            fmt.Println("found")
            index = j
            break
        }
    }
    if index == -1 {
        //file not found, stop
        fmt.Println("Could not find file " + file_name + " to delete")
        return errors.New("Could not delete template files.  One or more specified files does not exist")
    }    
    
    files = append(files[:index], files[index+1:]...)

    releases := t.GetTemplateCommitReleases()
    fmt.Println("releases len:" + strconv.Itoa(len(releases)))
    var err error
    for _, template := range releases {
       template.Template_hash = GenerateHashString()
       template.Commit_info = CommitInfo{"nguyener", time.Now().String(), "Delete template files", "Delete template files"}
       template.Files = files
       if err = template.CreateTemplate(); err != nil {
           return err
       }
    }
    return nil          
} 

//modify content fo file(s) under specified template commit (version).  A new template commit (version) will generated.  Affect all releases under the original commit
func (t *Template) ModifyTemplateFile(file_name string, modified_file File) error {
    files,_ := t.GetTemplateFiles()
    fmt.Println("files len: " + strconv.Itoa(len(files)))
        found := false
        for _, file := range files {
            fmt.Println("file.File_name is: " + file.File_name)
            if file.File_name == file_name {
                fmt.Println("found")
                file.File_content = modified_file.File_content
                found = true
                break
             }
         }
         if !found {
             fmt.Println("Could not find file " + file_name + " to modify")
             return errors.New("Could not modify template files.  One or more specified files does not exist")
         }
         
    
    
    releases := t.GetTemplateCommitReleases()
    var err error
    for _, template := range releases {
       template.Template_hash = GenerateHashString()
       template.Commit_info = CommitInfo{"nguyener", time.Now().String(), "Modify template files", "Modify template files"}
       template.Files = files
       if err = template.CreateTemplate(); err != nil {
           return err
       }
    }
    return nil
      
} 

//get all releases
func (t *Template) GetAllReleases() []Template {
    var release Template
    var releases []Template

    iter := Conn.Query(QUERY_STR_GET_ALL_RELEASES).Iter()
    for iter.Scan(&release.Template_id, &release.Template_name, &release.Template_hash, &release.Release_info){
        if &release.Release_info != nil  {
            releases = append(releases, release)
        }
    }
    return releases
}
/*
func (t *Template) GetAllReleases() []Template {
    var release Template
    var releases []Template

    iter := Conn.Query(QUERY_STR_GET_ALL_RELEASES).Iter()
    for iter.Scan(&release.Template_id, &release.Template_name, &release.Template_hash, &release.Release_info){
        if &release.Release_info != nil  {
            releases = append(releases, release)
        }
    }
    return releases
}
*/

//Get releases per template (all commits (versions)
func (t *Template) GetTemplateReleases() [] Template {
    /*
    var releases []Template
    all_releases := t.getAllReleases()
    
    for _, release := range all_releases {
        if &release.Release_info != nil && release.Template_name == t.Template_name {
            releases = append(releases, release)
        }
    }
    return releases
    */
    var release Template
    var releases []Template

    iter := Conn.Query(QUERY_STR_GET_ALL_RELEASES).Iter()
    for iter.Scan(&release.Template_id, &release.Template_name, &release.Template_hash, &release.Release_info){
        if &release.Release_info != nil && release.Template_name == t.Template_name {
            releases = append(releases, release)
        }
    }
    return releases

}

// get releases per template's commit (version)
func (t *Template) GetTemplateCommitReleases() [] Template {
   var release Template
   var releases []Template
    
   iter := Conn.Query(QUERY_STR_GET_ALL_RELEASES).Iter()
   for iter.Scan(&release.Template_id, &release.Template_name, &release.Template_hash, &release.Release_info){
       if &release.Release_info != nil && release.Template_hash == t.Template_hash {
           releases = append(releases, release)
       }   
   }   
   return releases
 
}



// create a new release for existing template commit (version).  A new row (new id, template_hash) will be inserted into templates table
func (t *Template) CreateTemplateRelease(release_info ReleaseInfo ) error {
    //duplicate the template commit (version) and assign release info
    templates := t.GetTemplateCommitReleases()
    found := false
    for _, template := range templates {
        if template.Release_info.Release_role == release_info.Release_role {
            found = true
            break
        }
    }
    if found {
        return errors.New("Specified release already exists")
    }
    
    t = &templates[0]
    t.Release_info = release_info
    
    return t.CreateTemplate()    
}


// delete a release row 
func (t *Template) DeleteTemplateRelease(release_role string ) error {
    //get the specified release row
    releases := t.GetTemplateCommitReleases()
    
    for _, release := range releases {
        if release.Release_info.Release_role == release_role {
            //found, delete now
            query_str := fmt.Sprintf(QUERY_STR_DELETE_TEMPLATE_ROWS, release.Template_id)
            err := Conn.Query(query_str).Exec()
            return err
        }
    }   
   
    return errors.New("Specified release does not already exist")
}


// modifiy release info
func (t *Template) ModifyTemplateRelease(release_role string, new_release_info ReleaseInfo ) error {
    //get the specified release row
    releases := t.GetTemplateCommitReleases()

    for _, release := range releases {
        if release.Release_info.Release_role == release_role {
            //found, update now
            query_str := fmt.Sprintf(QUERY_STR_UPDATE_RELEASE_INFO, new_release_info.toString(), release.Template_id)
            err := Conn.Query(query_str).Exec()
            return err
        }
    }

    return errors.New("Specified release does not already exist")
}




func (d *Device) GetAllDevices() []Device {
    var device Device
    var devices []Device
    iter := Conn.Query(QUERY_STR_GET_ALL_DEVICES).Iter()
    for iter.Scan(&device.Device_name, &device.Template_name, &device.Template_hash, &device.Release_info) {
        devices = append(devices, device)
    }
    return devices
}

func (d *Device) GetDevice() (*Device, error) {
    var device Device
    if d.Device_name == "" {
        return nil, errors.New("The device name is not provided")
    }

    query_str := fmt.Sprintf(QUERY_STR_GET_DEVICE_BY_NAME, d.Device_name)
    fmt.Println(query_str)
    if err := Conn.Query(fmt.Sprintf(QUERY_STR_GET_DEVICE_BY_NAME, d.Device_name)).Scan(&device.Device_name, &device.Template_name, &device.Template_hash, &device.Release_info); err != nil {
        return nil, err
    }else{
        return &device, nil
    }
}

func (d *Device) CreateDevice(payload Device) error {
    //check if the device has been registered
    d.Device_name = payload.Device_name
    device, _ := d.GetDevice()
      
    if device != nil {
        return errors.New("Specified device already exists")
    }
    
    if err := Conn.Query(fmt.Sprintf(QUERY_STR_INSERT_DEVICE, payload.Device_name, payload.Template_name, payload.Template_hash, payload.Release_info.toString())).Exec(); err != nil {

        return err
    }else{
        return nil
    }   
}  


func (d *Device) DeleteDevice() error {
    
     fmt.Println("in DeleteDevice")
    //check if the device has been registered
    device, err := d.GetDevice()
    if err != nil {
        fmt.Println(err)
        return err
    }
    
    fmt.Println("ok here")
    if device == nil {
        return errors.New("Specified device does not exist")
    }
    query_str := fmt.Sprintf(QUERY_STR_DELETE_DEVICE, d.Device_name)
    fmt.Println(query_str)
    if err = Conn.Query(fmt.Sprintf(QUERY_STR_DELETE_DEVICE, d.Device_name)).Exec(); err != nil {
        return err
    }else{
        return nil
    }
}

func (d *Device) UpdateDevice(payload Device) (error) {
    //check if the device has been registered
    device, err := d.GetDevice()
    if err != nil {
        return err
    }
    if device == nil {
        return errors.New("Specified device does not exist")
    }

    query_str := fmt.Sprintf(QUERY_STR_UPDATE_DEVICE, payload.Template_name, payload.Template_hash, payload.Release_info.toString(), d.Device_name)
    fmt.Println(query_str)
    if err := Conn.Query(fmt.Sprintf(QUERY_STR_UPDATE_DEVICE, payload.Template_name, payload.Template_hash, payload.Release_info.toString(), d.Device_name)).Exec(); err != nil {
        return err
    }else{
        return nil
    }

}   





   
func GenerateHashString() string {
    now :=time.Now()
    hasher := sha1.New()
    hasher.Write([]byte(now.String()))

    template_hash := fmt.Sprintf("%x", hasher.Sum(nil))
    return template_hash
}







