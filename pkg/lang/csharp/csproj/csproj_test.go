package csproj

import (
	"bytes"
	"strings"
	"testing"
)

func TestXmlPath(t *testing.T) {
	s := `
	
<Project Sdk="Microsoft.NET.Sdk.Web">
  <PropertyGroup>
    <TargetFramework>net6.0</TargetFramework>
    <GenerateRuntimeConfigurationFiles>true</GenerateRuntimeConfigurationFiles>
    <AWSProjectType>Lambda</AWSProjectType>
    <!-- This property makes the build directory similar to a publish directory and helps the AWS .NET Lambda Mock Test Tool find project dependencies. -->
    <CopyLocalLockFileAssemblies>true</CopyLocalLockFileAssemblies>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Amazon.Lambda.AspNetCoreServer" Version="7.2.0" />
  </ItemGroup>
</Project>
`
	f, _ := NewCSProjFile("file.csproj", strings.NewReader(s))
	f.AddProperty("OutDir", "klotho_bin")
	f.addKlothoProperties()
	//println(f.)
	b := bytes.NewBufferString("")
	f.WriteTo(b)
	println(b.String())
}
