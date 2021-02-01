package main

var xmlSetPrefixList = `
                <edit-config>
                        <target>
                                <candidate />
                        </target>
                        <default-operation>merge</default-operation>
                        <config>
                                <configuration junos:changed-seconds="1608304598" junos:changed-localtime="date-time">
                                        <policy-options>
                                                <prefix-list>
                                                        <name>prefix-name</name>
                                                        <placeholder-prefix-list-item>
                                                </prefix-list>
                                        </policy-options>
                                </configuration>
                        </config>
                </edit-config>
        </rpc>
]]>]]>
<?xml version="1.0" encoding="UTF-8"?>
<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" message-id="2">
  <commit/>
</rpc>
]]>]]>
<?xml version="1.0" encoding="UTF-8"?>
<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" message-id="3">
  <close-session/>
</rpc>
]]>]]>`


var xmlTermSetLicensed = `
                <edit-config>
                        <target>
                                <candidate />
                        </target>
                        <default-operation>merge</default-operation>
                        <config>
    				<configuration junos:changed-seconds="1608304598" junos:changed-localtime="date-time">
            				<firewall>
                				<family>
                    					<inet>
                        					<filter>
                            						<name>filter-name</name>
                            						<term>
                                						<name>licensed</name>
                                						<from-list>
                                						</from>
                            						</term>
                        					</filter>
                    					</inet>
                				</family>
            				</firewall>
    			</configuration>
                        </config>
                </edit-config>
        </rpc>
]]>]]>
<?xml version="1.0" encoding="UTF-8"?>
<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" message-id="2">
  <commit/>
</rpc>
]]>]]>
<?xml version="1.0" encoding="UTF-8"?>
<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" message-id="3">
  <close-session/>
</rpc>
]]>]]>`

var xmlTermSetNotLicensed = `
                <edit-config>
                        <target>
                                <candidate />
                        </target>
                        <default-operation>merge</default-operation>
                        <config>
                                <configuration junos:changed-seconds="1608304598" junos:changed-localtime="date-time">
                                        <firewall>
                                                <family>
                                                        <inet>
                                                                <filter>
                                                                        <name>filter-name</name>
                                                                        <term>
                                                                                <name>notlicensed</name>
                                                                                <from-list>
                                                                                </from>
                                                                        </term>
                                                                </filter>
                                                        </inet>
                                                </family>
                                        </firewall>
                        </configuration>
                        </config>
                </edit-config>
        </rpc>
]]>]]>
<?xml version="1.0" encoding="UTF-8"?>
<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" message-id="2">
  <commit/>
</rpc>
]]>]]>
<?xml version="1.0" encoding="UTF-8"?>
<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" message-id="3">
  <close-session/>
</rpc>
]]>]]>`


var xmlTermDeleteLicensed = `
                <edit-config>
                        <target>
                                <candidate />
                        </target>
                        <default-operation>merge</default-operation>
                        <config>
                                <configuration junos:changed-seconds="1608304598" junos:changed-localtime="date-time">
                                        <firewall>
                                                <family>
                                                        <inet>
                                                                <filter>
                                                                        <name>filter-name</name>
                                                                        <term>
                                                                                <name>licensed</name>
                                                                                <from-list>
                                                                                </from>
                                                                        </term>
                                                                </filter>
                                                        </inet>
                                                </family>
                                        </firewall>
                        </configuration>
                        </config>
                </edit-config>
        </rpc>
]]>]]>
<?xml version="1.0" encoding="UTF-8"?>
<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" message-id="2">
  <commit/>
</rpc>
]]>]]>
<?xml version="1.0" encoding="UTF-8"?>
<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" message-id="3">
  <close-session/>
</rpc>
]]>]]>`


var xmlTermDeleteNotLicensed = `
                <edit-config>
                        <target>
                                <candidate />
                        </target>
                        <default-operation>merge</default-operation>
                        <config>
                                <configuration junos:changed-seconds="1608304598" junos:changed-localtime="date-time">
                                        <firewall>
                                                <family>
                                                        <inet>
                                                                <filter>
                                                                        <name>filter-name</name>
                                                                        <term>
                                                                                <name>notlicensed</name>
                                                                                <from-list>
                                                                                </from>
                                                                        </term>
                                                                </filter>
                                                        </inet>
                                                </family>
                                        </firewall>
                        </configuration>
                        </config>
                </edit-config>
        </rpc>
]]>]]>
<?xml version="1.0" encoding="UTF-8"?>
<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" message-id="2">
  <commit/>
</rpc>
]]>]]>
<?xml version="1.0" encoding="UTF-8"?>
<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" message-id="3">
  <close-session/>
</rpc>
]]>]]>`


func GetXmlSetPrefixList() string {
	return xmlSetPrefixList
}

func GetXmlTermSetLicensed() string {
        return xmlTermSetLicensed
}

func GetXmlTermSetNotLicensed() string {
        return xmlTermSetNotLicensed
}

func GetXmlTermDeleteLicensed() string {
        return xmlTermDeleteLicensed
}

func GetXmlTermDeleteNotLicensed() string {
        return xmlTermDeleteNotLicensed
}

