package handlers

// skill_handler.go — 007 同步：文件系统技能激活块构建（Web 端等价实现）。
// 从上游桌面端根包 app.go 的 buildSkillContext 家族平移（桌面层被跳过、Web 端不能
// 反向依赖根包），逻辑与措辞保持逐行一致，确保桌面/Web 选中同一技能时注入的
// 系统提示词与激活块完全相同——技能效果不因端而异。
//
// 数据源与技能管理页（ListFilesystemSkills）一致：<agent.RootDir()/skills>/<dirName>/SKILL.md。

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"

	"go-stock/backend/agent"
	"go-stock/backend/logger"
)

// filesystemSkillInfo SKILL.md frontmatter 元信息。
type filesystemSkillInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	DirName     string `json:"dirName"`
}

// skillsDir 文件系统技能根目录（与桌面端 skillsDir() 同源）。
func skillsDir() string {
	return filepath.Join(agent.RootDir(), "skills")
}

// sanitizeSkillDirName 清洗技能目录名，禁止路径穿越。
func sanitizeSkillDirName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "..", "")
	name = strings.ReplaceAll(name, "/", "")
	name = strings.ReplaceAll(name, "\\", "")
	name = strings.ReplaceAll(name, ":", "")
	return strings.TrimSpace(name)
}

// parseSkillFrontmatter 解析 SKILL.md 的 YAML frontmatter（--- ... ---）。
func parseSkillFrontmatter(content string) filesystemSkillInfo {
	info := filesystemSkillInfo{}
	const delimiter = "---"
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, delimiter) {
		return info
	}
	rest := content[len(delimiter):]
	endIdx := strings.Index(rest, "\n"+delimiter)
	if endIdx == -1 {
		return info
	}
	frontmatter := strings.TrimSpace(rest[:endIdx])

	var fm struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}
	if err := yaml.Unmarshal([]byte(frontmatter), &fm); err == nil {
		info.Name = fm.Name
		info.Description = fm.Description
	}
	return info
}

// buildSkillContext 读取多个文件系统技能（逗号分隔目录名，支持多选），
// 合并构建系统提示词与用户消息激活块。单个技能读取失败时记录警告并跳过，不影响其他技能加载。
//
// 激活设计（提高技能实际生效成功率，尤其 DeepAgents 模式）：
//   - 系统提示词返回值：技能全文 + 激活纪律（强制主 Agent 应用方法论并在委派时传播技能要求）
//   - 问题块返回值：随用户消息提交。用户消息是唯一能经 task 委派描述传播到子 Agent 的通道
//     （子 Agent 不继承系统提示词），激活要求必须随问题文本下发才能触达子 Agent。
func buildSkillContext(dirNames string) (sysPrompt string, questionBlock string) {
	var sb strings.Builder
	var skillNames []string
	loaded := 0
	for _, name := range strings.Split(dirNames, ",") {
		name = sanitizeSkillDirName(name)
		if name == "" {
			continue
		}
		section, skillName, ok := buildSingleSkillSection(name)
		if !ok {
			logger.SugaredLogger.Warnf("技能 %s 加载失败（SKILL.md 读取失败或为空），已跳过", name)
			continue
		}
		if loaded == 0 {
			sb.WriteString("## 你具备以下专业技能：\n")
		}
		sb.WriteString(section)
		if skillName == "" {
			skillName = name
		}
		skillNames = append(skillNames, skillName)
		loaded++
	}
	if loaded == 0 {
		return "", ""
	}
	sb.WriteString(buildSkillDiscipline(skillNames))
	return sb.String(), buildSkillQuestionBlock(skillNames)
}

// buildSkillDiscipline 生成注入系统提示词的技能激活纪律。
// 技能全文注入系统提示词只代表主 Agent"具备"该知识，模型仍可能用通用泛泛分析替代；
// 纪律将"应用方法论"与"委派传播"变为强制要求。措辞对 Agent 模式自适应：
// 委派条款仅在环境提供 task 委派能力（DeepAgents）时被模型采用，React/PlanExecute 无 task 工具时自动忽略。
func buildSkillDiscipline(skillNames []string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n## 技能激活纪律（用户显式选择，必须遵守）\n用户在本次会话中显式选择了技能：%s。技能内容不是参考资料，而是本次分析的强制执行标准：\n\n", strings.Join(skillNames, "、")))
	sb.WriteString("1. 【立即应用】从第一条分析开始就必须使用上述技能的方法论、术语体系和输出框架，禁止用通用泛泛分析替代。\n")
	sb.WriteString("2. 【委派传播】若你具备 task 委派能力（DeepAgents 模式），委派任何子任务时，description 必须原样包含所需技能名称，并明确要求子 Agent 先调用 skill 工具加载该技能、严格按其方法论执行。禁止委派不带技能要求的分析任务。\n")
	sb.WriteString("3. 【输出自查】最终输出前自查：结论是否呈现技能方法论的特征步骤（该技能定义的分析框架、清单、评级体系等）。若未体现，必须重新按技能流程执行后再输出。\n")
	return sb.String()
}

// buildSkillQuestionBlock 生成注入用户消息（问题文本前）的技能激活块。
// 用户消息会随 task 委派描述被主 Agent 提炼传播给子 Agent，是触达子 Agent 的关键通道；
// 块中技能名称必须与 skill 工具 <name> 参数一致（均取自 SKILL.md frontmatter name）。
func buildSkillQuestionBlock(skillNames []string) string {
	var sb strings.Builder
	sb.WriteString("<selected_skills>\n")
	sb.WriteString("用户已显式选择以下技能，本次分析必须使用（不是可选参考）：\n")
	for i, name := range skillNames {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, name))
	}
	sb.WriteString("\n执行纪律：\n")
	sb.WriteString("- 全程严格遵循技能的方法论与输出格式，最终结论必须体现技能的分析框架\n")
	sb.WriteString("- 若委派子任务（task 工具可用时），任务描述必须原样包含技能名称，并要求子 Agent 先调用 skill 工具加载该技能后再执行\n")
	sb.WriteString("</selected_skills>\n\n")
	return sb.String()
}

// buildSingleSkillSection 读取单个技能的 SKILL.md，剥离 frontmatter 后返回该技能的提示词片段与 frontmatter 名称。
// 与技能管理页面（ListFilesystemSkills）数据源一致，确保 / 斜杠指令选择的就是用户在技能管理中看到的技能。
// 名称用于激活纪律与 skill 工具参数匹配（skill 工具按 frontmatter name 精确匹配）。
func buildSingleSkillSection(dirName string) (section string, skillName string, ok bool) {
	skillMdPath := filepath.Join(skillsDir(), dirName, "SKILL.md")
	data, err := os.ReadFile(skillMdPath)
	if err != nil {
		return "", "", false
	}
	content := strings.TrimSpace(string(data))
	if content == "" {
		return "", "", false
	}
	info := parseSkillFrontmatter(content)
	// 剥离 frontmatter（--- ... ---），取正文指令
	const delimiter = "---"
	if strings.HasPrefix(content, delimiter) {
		rest := content[len(delimiter):]
		endIdx := strings.Index(rest, "\n"+delimiter)
		if endIdx != -1 {
			body := strings.TrimSpace(rest[endIdx+len(delimiter)+2:])
			var sb strings.Builder
			if info.Name != "" {
				sb.WriteString(fmt.Sprintf("\n### %s\n", info.Name))
			}
			if info.Description != "" {
				sb.WriteString(info.Description + "\n")
			}
			if body != "" {
				sb.WriteString(body + "\n")
			}
			return sb.String(), info.Name, true
		}
	}
	// 无 frontmatter：全文即指令
	return content, info.Name, true
}

// skillMdFileName / skillMdDisabledName 启用/停用态的技能定义文件名（与桌面端一致）。
const (
	skillMdFileName       = "SKILL.md"
	skillMdDisabledName   = "SKILL.md.disabled"
)

// ListFilesystemSkillsHandler GET /skills/filesystem 列出服务器端文件系统技能。
// 与桌面端 Wails 绑定同语义：停用技能带 disabled 标志返回（技能管理页展示用），
// 前端 Agent 聊天侧（斜杠技能菜单）需按 disabled 过滤。
func ListFilesystemSkillsHandler(c *gin.Context) {
	dir := skillsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": []gin.H{}})
		return
	}
	type fsSkill struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		DirName     string `json:"dirName"`
		Disabled    bool   `json:"disabled"`
	}
	var result []fsSkill
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		var data []byte
		disabled := false
		enabledPath := filepath.Join(dir, entry.Name(), skillMdFileName)
		if _, statErr := os.Stat(enabledPath); statErr == nil {
			data, _ = os.ReadFile(enabledPath)
		} else {
			disabledPath := filepath.Join(dir, entry.Name(), skillMdDisabledName)
			if _, statErr := os.Stat(disabledPath); statErr == nil {
				data, _ = os.ReadFile(disabledPath)
				disabled = true
			} else {
				continue
			}
		}
		if len(data) == 0 {
			continue
		}
		info := parseSkillFrontmatter(string(data))
		result = append(result, fsSkill{
			Name:        info.Name,
			Description: info.Description,
			DirName:     entry.Name(),
			Disabled:    disabled,
		})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": result})
}
